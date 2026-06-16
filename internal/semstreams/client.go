package semstreams

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/c360studio/semlink/internal/graphprojection"
	"github.com/c360studio/semstreams/graph"
	"github.com/c360studio/semstreams/natsclient"
	graphingest "github.com/c360studio/semstreams/processor/graph-ingest"
)

type requester interface {
	Request(ctx context.Context, subject string, data []byte, timeout time.Duration) ([]byte, error)
}

type GraphClient struct {
	requester requester
	timeout   time.Duration
	mu        sync.RWMutex
	known     map[string]struct{}
}

type WriteResult struct {
	EntityID   string `json:"entity_id"`
	Revision   uint64 `json:"revision"`
	Created    bool   `json:"created"`
	Triples    int    `json:"triples"`
	Profile    string `json:"profile"`
	Degraded   bool   `json:"degraded"`
	SemStreams bool   `json:"semstreams"`
}

func NewGraphClient(requester requester, timeout time.Duration) *GraphClient {
	if timeout <= 0 {
		timeout = 5 * time.Second
	}
	return &GraphClient{
		requester: requester,
		timeout:   timeout,
		known:     make(map[string]struct{}),
	}
}

func (c *GraphClient) UpsertProjection(ctx context.Context, p graphprojection.Projection) (*WriteResult, error) {
	if p.Entity == nil {
		return nil, errors.New("semstreams graph upsert: nil entity")
	}
	if len(p.Triples) == 0 {
		return nil, errors.New("semstreams graph upsert: no triples")
	}

	if c.isKnown(p.Entity.ID) {
		result, err := c.updateProjection(ctx, p)
		if err == nil {
			return result, nil
		}
		if !errors.Is(err, errEntityNotFound) {
			return nil, err
		}
		c.forget(p.Entity.ID)
	}

	return c.createProjection(ctx, p)
}

var errEntityNotFound = errors.New("semstreams entity not found")

func (c *GraphClient) createProjection(ctx context.Context, p graphprojection.Projection) (*WriteResult, error) {
	createReq := graph.CreateEntityWithTriplesRequest{
		Entity:          p.Entity,
		Triples:         p.Triples,
		IndexingProfile: p.IndexingProfile,
		RequestID:       fmt.Sprintf("semlink-create-%d", time.Now().UnixNano()),
	}
	createResp, err := requestJSON[graph.CreateEntityWithTriplesResponse](ctx, c, graphingest.SubjectEntityCreateWithTriples, createReq)
	if err != nil {
		return nil, err
	}
	if createResp.Success || createResp.Degraded {
		c.remember(p.Entity.ID)
		return &WriteResult{
			EntityID:   p.Entity.ID,
			Revision:   createResp.KVRevision,
			Created:    true,
			Triples:    createResp.TriplesAdded,
			Profile:    p.IndexingProfile,
			Degraded:   createResp.Degraded,
			SemStreams: true,
		}, nil
	}
	if createResp.ErrorCode != graph.ErrorCodeEntityExists {
		return nil, fmt.Errorf("semstreams create %s rejected: %s", p.Entity.ID, createResp.Error)
	}

	c.remember(p.Entity.ID)
	return c.updateProjection(ctx, p)
}

func (c *GraphClient) updateProjection(ctx context.Context, p graphprojection.Projection) (*WriteResult, error) {
	updateReq := graph.UpdateEntityWithTriplesRequest{
		Entity: &graph.EntityState{
			ID:          p.Entity.ID,
			MessageType: p.Entity.MessageType,
			UpdatedAt:   p.Entity.UpdatedAt,
		},
		AddTriples: p.Triples,
		RequestID:  fmt.Sprintf("semlink-update-%d", time.Now().UnixNano()),
	}
	updateResp, err := requestJSON[graph.UpdateEntityWithTriplesResponse](ctx, c, graphingest.SubjectEntityUpdateWithTriples, updateReq)
	if err != nil {
		return nil, err
	}
	if updateResp.Success || updateResp.Degraded {
		c.remember(p.Entity.ID)
		return &WriteResult{
			EntityID:   p.Entity.ID,
			Revision:   updateResp.KVRevision,
			Created:    false,
			Triples:    updateResp.TriplesAdded,
			Profile:    p.IndexingProfile,
			Degraded:   updateResp.Degraded,
			SemStreams: true,
		}, nil
	}
	if updateResp.ErrorCode == graph.ErrorCodeEntityNotFound {
		return nil, errEntityNotFound
	}
	return nil, fmt.Errorf("semstreams update %s rejected: %s", p.Entity.ID, updateResp.Error)
}

func (c *GraphClient) isKnown(id string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	_, ok := c.known[id]
	return ok
}

func (c *GraphClient) remember(id string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.known[id] = struct{}{}
}

func (c *GraphClient) forget(id string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.known, id)
}

func (c *GraphClient) QueryEntity(ctx context.Context, id string) (*graph.EntityState, error) {
	req := struct {
		ID string `json:"id"`
	}{ID: id}
	data, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	resp, err := c.requester.Request(ctx, "graph.ingest.query.entity", data, c.timeout)
	if err != nil {
		return nil, err
	}
	var entity graph.EntityState
	if err := json.Unmarshal(resp, &entity); err != nil {
		return nil, err
	}
	return &entity, nil
}

func requestJSON[T any](ctx context.Context, c *GraphClient, subject string, req any) (T, error) {
	var zero T
	data, err := json.Marshal(req)
	if err != nil {
		return zero, err
	}
	resp, err := c.requester.Request(ctx, subject, data, c.timeout)
	if err != nil {
		return zero, err
	}
	var out T
	if err := json.Unmarshal(resp, &out); err != nil {
		return zero, err
	}
	return out, nil
}

func PublishRawFrame(ctx context.Context, client *natsclient.Client, subject string, data []byte) error {
	_, err := client.PublishToStreamWithAck(ctx, subject, data)
	return err
}

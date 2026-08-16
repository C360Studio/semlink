package gcs

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/c360studio/semlink/internal/projector"
	semruntime "github.com/c360studio/semlink/internal/semstreams"
)

type CommandService struct {
	graph *semruntime.GraphClient
	store *Store
}

func NewCommandService(graph *semruntime.GraphClient, store *Store) *CommandService {
	return &CommandService{graph: graph, store: store}
}

func (s *CommandService) Submit(ctx context.Context, targetEntity, verb string) (CommandView, error) {
	systemID, err := systemIDFromEntity(targetEntity)
	if err != nil {
		return CommandView{}, err
	}
	verb = normalizeCommand(verb)
	if verb == "" {
		return CommandView{}, fmt.Errorf("empty command verb")
	}
	now := time.Now()
	payload := projector.CommandPayload{
		ID:           projector.CommandEntityID(verb, systemID, now),
		TargetEntity: targetEntity,
		Verb:         verb,
		Status:       "requested",
		RequestedAt:  now,
	}
	proj := projector.ProjectionFromPayload(&payload, projector.CommandType, now)
	start := time.Now()
	result, err := s.graph.UpsertProjection(ctx, proj)
	if err != nil {
		return CommandView{}, err
	}
	cmd := CommandView{
		EntityID:      payload.ID,
		TargetEntity:  targetEntity,
		Verb:          verb,
		Status:        payload.Status,
		RequestedAt:   now,
		GraphRevision: result.Revision,
	}
	s.store.AddCommand(cmd)
	s.store.RecordGraphWrite(payload.ID, result.Revision, time.Since(start))
	return cmd, nil
}

func normalizeCommand(verb string) string {
	return strings.ReplaceAll(strings.ToLower(strings.TrimSpace(verb)), "_", "-")
}

func systemIDFromEntity(entity string) (uint8, error) {
	idx := strings.LastIndex(entity, "uav-")
	if idx == -1 {
		return 0, fmt.Errorf("target entity %q does not contain uav id", entity)
	}
	raw := entity[idx+4:]
	if dash := strings.IndexByte(raw, '-'); dash >= 0 {
		raw = raw[:dash]
	}
	id, err := strconv.Atoi(raw)
	if err != nil || id < 1 || id > 255 {
		return 0, fmt.Errorf("invalid uav id in target entity %q", entity)
	}
	return uint8(id), nil
}

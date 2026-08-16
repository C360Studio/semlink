package semstreams

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"sync/atomic"
	"time"

	"github.com/c360studio/semlink/internal/graphprojection"
	"github.com/c360studio/semstreams/graph"
	"github.com/c360studio/semstreams/message"
	"github.com/c360studio/semstreams/natsclient"
	semerrs "github.com/c360studio/semstreams/pkg/errs"
	"github.com/c360studio/semstreams/pkg/projection"
)

const (
	defaultMutationAttempts = 3
	createEnvelopeSource    = "semlink.graph-adapter"
)

type mutationPort interface {
	projection.EntityCreator
	projection.PredicateReconciler
	projection.AuthoritativeReader
}

type GraphClient struct {
	mutations       mutationPort
	groupPredicates map[string]map[string]map[string]struct{}
	maxAttempts     int
	nextRequestID   func() string
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

var requestSequence atomic.Uint64

// NewGraphClient constructs the canonical beta.160 mutation adapter from the
// complete locally validated projection contract set.
func NewGraphClient(client *natsclient.Client, contracts []projection.Contract, timeout time.Duration) (*GraphClient, error) {
	mutationClient, err := projection.NewMutationClient(projection.MutationClientConfig{
		NATS:      client,
		Contracts: contracts,
		Timeout:   timeout,
	})
	if err != nil {
		return nil, fmt.Errorf("create SemStreams mutation client: %w", err)
	}
	return newGraphClient(mutationClient, contracts, defaultMutationAttempts, defaultRequestID), nil
}

func newGraphClient(port mutationPort, contracts []projection.Contract, maxAttempts int, nextRequestID func() string) *GraphClient {
	if maxAttempts <= 0 {
		maxAttempts = defaultMutationAttempts
	}
	if nextRequestID == nil {
		nextRequestID = defaultRequestID
	}
	groups := make(map[string]map[string]map[string]struct{}, len(contracts))
	for _, contract := range contracts {
		contractGroups := make(map[string]map[string]struct{}, len(contract.Groups))
		for _, group := range contract.Groups {
			predicates := make(map[string]struct{}, len(group.Predicates))
			for _, predicate := range group.Predicates {
				predicates[predicate] = struct{}{}
			}
			contractGroups[group.Name] = predicates
		}
		groups[contract.Name] = contractGroups
	}
	return &GraphClient{
		mutations:       port,
		groupPredicates: groups,
		maxAttempts:     maxAttempts,
		nextRequestID:   nextRequestID,
	}
}

func defaultRequestID() string {
	return fmt.Sprintf("semlink-%d-%d", time.Now().UnixNano(), requestSequence.Add(1))
}

// UpsertProjection reconciles a complete named group. A missing entity is
// established with an empty strict-create envelope before an immediate retry.
func (c *GraphClient) UpsertProjection(ctx context.Context, value graphprojection.Projection) (*WriteResult, error) {
	if c == nil || c.mutations == nil {
		return nil, errors.New("semstreams graph upsert: mutation client unavailable")
	}
	if value.Entity == nil {
		return nil, errors.New("semstreams graph upsert: nil entity")
	}
	if len(value.Triples) == 0 {
		return nil, errors.New("semstreams graph upsert: no triples")
	}
	if value.Contract == "" || value.Group == "" {
		return nil, errors.New("semstreams graph upsert: projection contract and group are required")
	}
	if _, ok := c.groupPredicates[value.Contract][value.Group]; !ok {
		return nil, fmt.Errorf("semstreams graph upsert: unknown projection binding %q/%q", value.Contract, value.Group)
	}

	requestID := c.nextRequestID()
	createMetadata := projection.MutationMetadata{
		RequestID: requestID + "-create",
		TraceID:   requestID,
		Source:    createEnvelopeSource,
		Timestamp: time.Now().UTC(),
	}
	created := false
	var lastErr error
	for attempt := 0; attempt < c.maxAttempts; attempt++ {
		receipt, err := c.reconcileProjection(ctx, value, requestID)
		if err == nil {
			if err := validateVerifiedReceipt(projection.MutationOperationReconcile, receipt); err != nil {
				return nil, err
			}
			return writeResult(value, receipt.KVRevision, created), nil
		}
		lastErr = err

		mutationErr := asMutationError(err)
		if mutationErr == nil {
			return nil, err
		}
		switch {
		case mutationErr.Kind == projection.MutationRevisionConflict && mutationErr.Commit == projection.CommitNotCommitted:
			continue
		case mutationErr.Kind == projection.MutationCommitUnknown || mutationErr.Commit == projection.CommitUnknown:
			exact, readErr := c.mutations.ReadAuthoritative(ctx, value.Entity.ID)
			if readErr == nil {
				if c.projectionMatches(exact, value, requestID) {
					return writeResult(value, exact.KVRevision, created), nil
				}
				continue
			}
			if !isTyped(readErr, projection.MutationNotFound, projection.CommitNotCommitted) {
				return nil, fmt.Errorf("resolve uncertain reconcile: %w", readErr)
			}
			// An authoritative absence proves the uncertain reconcile did not
			// establish the entity, so the safe birth path may proceed.
		case mutationErr.Kind == projection.MutationNotFound && mutationErr.Commit == projection.CommitNotCommitted:
			// Proceed to strict envelope creation below.
		default:
			return nil, err
		}

		createReceipt, createErr := c.mutations.Create(ctx, projection.CreateMutation{
			Contract: value.Contract,
			Entity:   entityEnvelope(value.Entity),
			Triples:  nil,
			Metadata: createMetadata,
		})
		entityAvailable := false
		if createErr == nil {
			if err := validateVerifiedReceipt(projection.MutationOperationCreate, createReceipt); err != nil {
				return nil, err
			}
			created = true
			entityAvailable = true
		} else {
			lastErr = createErr
			createMutationErr := asMutationError(createErr)
			if createMutationErr == nil {
				return nil, createErr
			}
			switch {
			case createMutationErr.Kind == projection.MutationConflict && createMutationErr.Commit == projection.CommitNotCommitted:
				entityAvailable = true
			case createMutationErr.Kind == projection.MutationCommitUnknown || createMutationErr.Commit == projection.CommitUnknown:
				exact, readErr := c.mutations.ReadAuthoritative(ctx, value.Entity.ID)
				if readErr == nil && exact != nil && exact.Entity != nil {
					entityAvailable = true
					break
				}
				if isTyped(readErr, projection.MutationNotFound, projection.CommitNotCommitted) {
					continue
				}
				if readErr != nil {
					return nil, fmt.Errorf("resolve uncertain create: %w", readErr)
				}
				return nil, createErr
			default:
				return nil, createErr
			}
		}

		if entityAvailable {
			// Entity birth and create-conflict convergence always receive an
			// immediate group reconcile, even when the normal attempt budget is
			// exhausted. This is one bounded companion operation, not an
			// unbounded retry.
			immediate, immediateErr := c.reconcileProjection(ctx, value, requestID)
			if immediateErr == nil {
				if err := validateVerifiedReceipt(projection.MutationOperationReconcile, immediate); err != nil {
					return nil, err
				}
				return writeResult(value, immediate.KVRevision, created), nil
			}
			lastErr = immediateErr
			immediateMutationErr := asMutationError(immediateErr)
			if immediateMutationErr == nil {
				return nil, immediateErr
			}
			if immediateMutationErr.Kind == projection.MutationCommitUnknown || immediateMutationErr.Commit == projection.CommitUnknown {
				exact, readErr := c.mutations.ReadAuthoritative(ctx, value.Entity.ID)
				if readErr == nil && c.projectionMatches(exact, value, requestID) {
					return writeResult(value, exact.KVRevision, created), nil
				}
				if readErr != nil && !isTyped(readErr, projection.MutationNotFound, projection.CommitNotCommitted) {
					return nil, fmt.Errorf("resolve uncertain immediate reconcile: %w", readErr)
				}
			}
			if attempt+1 < c.maxAttempts && (immediateMutationErr.Kind == projection.MutationRevisionConflict ||
				immediateMutationErr.Kind == projection.MutationNotFound ||
				immediateMutationErr.Kind == projection.MutationCommitUnknown) {
				continue
			}
			return nil, immediateErr
		}
	}
	return nil, lastErr
}

func (c *GraphClient) reconcileProjection(ctx context.Context, value graphprojection.Projection, requestID string) (projection.MutationReceipt, error) {
	return c.mutations.Reconcile(ctx, projection.ReconcileMutation{
		Contract: value.Contract,
		Group:    value.Group,
		EntityID: value.Entity.ID,
		Desired:  cloneTriples(value.Triples),
		Metadata: projection.MutationMetadata{RequestID: requestID, TraceID: requestID},
	})
}

func (c *GraphClient) QueryExactEntity(ctx context.Context, entityID string) (*graph.ExactEntity, error) {
	if c == nil || c.mutations == nil {
		return nil, errors.New("semstreams exact entity read: mutation client unavailable")
	}
	exact, err := c.mutations.ReadAuthoritative(ctx, entityID)
	if err != nil {
		return nil, err
	}
	if exact == nil || exact.Entity == nil || exact.KVRevision == 0 {
		return nil, errors.New("semstreams exact entity read: invalid authoritative result")
	}
	return &graph.ExactEntity{Entity: exact.Entity.Clone(), KVRevision: exact.KVRevision}, nil
}

func (c *GraphClient) QueryEntity(ctx context.Context, entityID string) (*graph.EntityState, error) {
	exact, err := c.QueryExactEntity(ctx, entityID)
	if err != nil {
		return nil, err
	}
	return exact.Entity.Clone(), nil
}

func (c *GraphClient) projectionMatches(exact *graph.ExactEntity, value graphprojection.Projection, requestID string) bool {
	if exact == nil || exact.Entity == nil || exact.Entity.ID != value.Entity.ID || exact.KVRevision == 0 {
		return false
	}
	group := c.groupPredicates[value.Contract][value.Group]
	actual := make([]message.Triple, 0, len(value.Triples))
	for _, triple := range exact.Entity.Triples {
		if _, ok := group[triple.Predicate]; ok {
			actual = append(actual, triple)
		}
	}
	expected := cloneTriples(value.Triples)
	for index := range expected {
		if expected[index].Context == "" {
			expected[index].Context = requestID
		}
	}
	return equalTripleSets(actual, expected)
}

func equalTripleSets(left, right []message.Triple) bool {
	if len(left) != len(right) {
		return false
	}
	canonical := func(triples []message.Triple) ([]string, bool) {
		out := make([]string, len(triples))
		for index, triple := range triples {
			encoded, err := json.Marshal(triple)
			if err != nil {
				return nil, false
			}
			out[index] = string(encoded)
		}
		sort.Strings(out)
		return out, true
	}
	leftValues, leftOK := canonical(left)
	rightValues, rightOK := canonical(right)
	if !leftOK || !rightOK {
		return false
	}
	for index := range leftValues {
		if leftValues[index] != rightValues[index] {
			return false
		}
	}
	return true
}

func entityEnvelope(entity *graph.EntityState) *graph.EntityState {
	envelope := entity.Clone()
	envelope.Triples = nil
	return envelope
}

func cloneTriples(triples []message.Triple) []message.Triple {
	return append([]message.Triple(nil), triples...)
}

func writeResult(value graphprojection.Projection, revision uint64, created bool) *WriteResult {
	return &WriteResult{
		EntityID:   value.Entity.ID,
		Revision:   revision,
		Created:    created,
		Triples:    len(value.Triples),
		Profile:    value.IndexingProfile,
		SemStreams: true,
	}
}

func asMutationError(err error) *projection.MutationError {
	var mutationErr *projection.MutationError
	if errors.As(err, &mutationErr) {
		return mutationErr
	}
	return nil
}

func isTyped(err error, kind projection.MutationErrorKind, commit projection.CommitState) bool {
	mutationErr := asMutationError(err)
	return mutationErr != nil && mutationErr.Kind == kind && mutationErr.Commit == commit
}

func unverifiedReceiptError(operation projection.MutationOperation, commit projection.CommitState) error {
	kind := projection.MutationInternal
	if commit == projection.CommitUnknown {
		kind = projection.MutationCommitUnknown
	}
	return &projection.MutationError{
		Operation: operation,
		Kind:      kind,
		Class:     semerrs.ErrorFatal,
		Commit:    commit,
		Err:       errors.New("mutation returned no verified commitment"),
	}
}

func validateVerifiedReceipt(operation projection.MutationOperation, receipt projection.MutationReceipt) error {
	if receipt.Commit != projection.CommitVerified {
		return unverifiedReceiptError(operation, receipt.Commit)
	}
	if receipt.KVRevision == 0 {
		return &projection.MutationError{
			Operation: operation,
			Kind:      projection.MutationInternal,
			Class:     semerrs.ErrorFatal,
			Commit:    projection.CommitUnknown,
			Err:       errors.New("verified mutation returned zero authoritative revision"),
		}
	}
	return nil
}

func PublishRawFrame(ctx context.Context, client *natsclient.Client, subject string, data []byte) error {
	_, err := client.PublishToStreamWithAck(ctx, subject, data)
	return err
}

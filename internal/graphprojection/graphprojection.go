package graphprojection

import (
	"time"

	"github.com/c360studio/semstreams/graph"
	"github.com/c360studio/semstreams/message"
)

// Projection is one atomic graph write through SemStreams graph-ingest.
type Projection struct {
	Entity          *graph.EntityState
	Triples         []message.Triple
	IndexingProfile string
	Contract        string
	Group           string
}

// Payload is the source-neutral payload contract needed to project an entity.
type Payload interface {
	message.Payload
	EntityID() string
	Triples() []message.Triple
	IndexingProfile() string
}

func Triple(subject, predicate string, object any, source string, timestamp time.Time, confidence float64) message.Triple {
	return message.Triple{
		Subject:    subject,
		Predicate:  predicate,
		Object:     object,
		Source:     source,
		Timestamp:  timestamp,
		Confidence: confidence,
	}
}

func ProjectionFromPayload(payload Payload, msgType message.Type, updatedAt time.Time, contract, group string) Projection {
	return Projection{
		Entity: &graph.EntityState{
			ID:          payload.EntityID(),
			MessageType: msgType,
			UpdatedAt:   updatedAt,
		},
		Triples:         payload.Triples(),
		IndexingProfile: payload.IndexingProfile(),
		Contract:        contract,
		Group:           group,
	}
}

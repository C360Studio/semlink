package mesh

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

type SourceKind string

const (
	SourceKindMAVLinkProjection SourceKind = "mavlink-projection"
	SourceKindPeerSummary       SourceKind = "peer-summary"
	SourceKindOperator          SourceKind = "operator"
	SourceKindRuleEngine        SourceKind = "rule-engine"
)

type MergePolicy string

const (
	MergePolicyLastWriterWins   MergePolicy = "last-writer-wins"
	MergePolicySetUnion         MergePolicy = "set-union"
	MergePolicyBoundedTombstone MergePolicy = "bounded-tombstone"
	MergePolicyAppendLimited    MergePolicy = "append-limited"
	MergePolicyCommandEvidence  MergePolicy = "command-evidence"
)

type HybridLogicalTime struct {
	WallTime time.Time `json:"wall_time"`
	Counter  uint64    `json:"counter"`
}

func (h HybridLogicalTime) IsZero() bool {
	return h.WallTime.IsZero() && h.Counter == 0
}

type SemStreamsEvidence struct {
	KVRevision      uint64    `json:"kv_revision,omitempty"`
	EntityVersion   uint64    `json:"entity_version,omitempty"`
	EntityUpdatedAt time.Time `json:"entity_updated_at,omitempty"`
	TripleTimestamp time.Time `json:"triple_timestamp,omitempty"`
}

// Envelope carries mesh-level causality and merge metadata. SemStreamsEvidence
// is intentionally evidence only; it is not accepted as distributed ordering.
type Envelope struct {
	OriginNodeID    string              `json:"origin_node_id"`
	OriginVehicleID string              `json:"origin_vehicle_id"`
	EntityID        string              `json:"entity_id"`
	PredicateGroup  string              `json:"predicate_group"`
	OriginSequence  uint64              `json:"origin_sequence,omitempty"`
	HybridTime      *HybridLogicalTime  `json:"hybrid_logical_time,omitempty"`
	OperationID     string              `json:"operation_id"`
	ObservedAt      time.Time           `json:"observed_at"`
	ExpiresAt       time.Time           `json:"expires_at"`
	SourceKind      SourceKind          `json:"source_kind"`
	Confidence      float64             `json:"confidence"`
	MergePolicy     MergePolicy         `json:"merge_policy"`
	PayloadHash     string              `json:"payload_hash"`
	SemStreams      *SemStreamsEvidence `json:"semstreams,omitempty"`
}

type Item struct {
	Envelope Envelope        `json:"envelope"`
	Payload  json.RawMessage `json:"payload"`
}

func NewItem(envelope Envelope, payload any) (Item, error) {
	raw, err := json.Marshal(payload)
	if err != nil {
		return Item{}, fmt.Errorf("marshal mesh payload: %w", err)
	}
	hash := HashBytes(raw)
	if envelope.PayloadHash == "" {
		envelope.PayloadHash = hash
	} else if envelope.PayloadHash != hash {
		return Item{}, fmt.Errorf("payload hash mismatch: envelope %s computed %s", envelope.PayloadHash, hash)
	}
	item := Item{Envelope: envelope, Payload: raw}
	if err := item.Validate(); err != nil {
		return Item{}, err
	}
	return item, nil
}

func (i Item) Validate() error {
	if err := i.Envelope.Validate(); err != nil {
		return err
	}
	if len(i.Payload) == 0 {
		return errors.New("mesh payload is required")
	}
	if got := HashBytes(i.Payload); got != i.Envelope.PayloadHash {
		return fmt.Errorf("payload hash mismatch: envelope %s computed %s", i.Envelope.PayloadHash, got)
	}
	return nil
}

func (e Envelope) Validate() error {
	var missing []string
	for name, value := range map[string]string{
		"origin_node_id":    e.OriginNodeID,
		"origin_vehicle_id": e.OriginVehicleID,
		"entity_id":         e.EntityID,
		"predicate_group":   e.PredicateGroup,
		"operation_id":      e.OperationID,
		"payload_hash":      e.PayloadHash,
	} {
		if strings.TrimSpace(value) == "" {
			missing = append(missing, name)
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("mesh envelope missing %s", strings.Join(missing, ", "))
	}
	if e.OriginSequence == 0 && (e.HybridTime == nil || e.HybridTime.IsZero()) {
		return errors.New("mesh envelope requires origin sequence or hybrid logical time")
	}
	if e.ObservedAt.IsZero() {
		return errors.New("mesh envelope observed_at is required")
	}
	if e.ExpiresAt.IsZero() {
		return errors.New("mesh envelope expires_at is required")
	}
	if !e.ExpiresAt.After(e.ObservedAt) {
		return errors.New("mesh envelope expires_at must be after observed_at")
	}
	if e.Confidence < 0 || e.Confidence > 1 {
		return fmt.Errorf("mesh envelope confidence out of range: %f", e.Confidence)
	}
	if !e.SourceKind.Valid() {
		return fmt.Errorf("mesh envelope source_kind %q is not supported", e.SourceKind)
	}
	if !e.MergePolicy.Valid() {
		return fmt.Errorf("mesh envelope merge_policy %q is not supported", e.MergePolicy)
	}
	if !strings.HasPrefix(e.PayloadHash, "sha256:") || len(e.PayloadHash) != len("sha256:")+sha256.Size*2 {
		return fmt.Errorf("mesh envelope payload_hash must be sha256:<hex>, got %q", e.PayloadHash)
	}
	return nil
}

func (s SourceKind) Valid() bool {
	switch s {
	case SourceKindMAVLinkProjection, SourceKindPeerSummary, SourceKindOperator, SourceKindRuleEngine:
		return true
	default:
		return false
	}
}

func (m MergePolicy) Valid() bool {
	switch m {
	case MergePolicyLastWriterWins, MergePolicySetUnion, MergePolicyBoundedTombstone, MergePolicyAppendLimited, MergePolicyCommandEvidence:
		return true
	default:
		return false
	}
}

func HashPayload(payload any) (string, error) {
	raw, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("marshal mesh payload for hash: %w", err)
	}
	return HashBytes(raw), nil
}

func HashBytes(raw []byte) string {
	sum := sha256.Sum256(raw)
	return "sha256:" + hex.EncodeToString(sum[:])
}

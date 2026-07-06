package rules

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/c360studio/semlink/internal/graphprojection"
	"github.com/c360studio/semstreams/message"
	"github.com/c360studio/semstreams/payloadregistry"
	"github.com/c360studio/semstreams/vocabulary"
)

type semType = message.Type

var TraceType = mustType("rules", "trace", "v1")

type InputScope string

const (
	InputScopeLocal InputScope = "local"
	InputScopeMesh  InputScope = "mesh"
)

type ExecutionPosture string

const ExecutionPostureObserveOnly ExecutionPosture = "observe-only"

type InputFact struct {
	Scope      InputScope `json:"scope"`
	Subject    string     `json:"subject"`
	Predicate  string     `json:"predicate"`
	Object     any        `json:"object"`
	Source     string     `json:"source"`
	ObservedAt time.Time  `json:"observed_at"`
	Confidence float64    `json:"confidence"`
}

func (f InputFact) Validate() error {
	if !f.Scope.Valid() {
		return fmt.Errorf("rule input fact scope %q is not supported", f.Scope)
	}
	if strings.TrimSpace(f.Subject) == "" || strings.TrimSpace(f.Predicate) == "" || strings.TrimSpace(f.Source) == "" {
		return errors.New("rule input fact subject, predicate, and source are required")
	}
	if f.ObservedAt.IsZero() {
		return errors.New("rule input fact observed_at is required")
	}
	if f.Confidence < 0 || f.Confidence > 1 {
		return fmt.Errorf("rule input fact confidence out of range: %f", f.Confidence)
	}
	if _, err := json.Marshal(f.Object); err != nil {
		return fmt.Errorf("marshal rule input fact object: %w", err)
	}
	return nil
}

func (s InputScope) Valid() bool {
	switch s {
	case InputScopeLocal, InputScopeMesh:
		return true
	default:
		return false
	}
}

type EvaluationInput struct {
	NodeID      string      `json:"node_id"`
	VehicleID   string      `json:"vehicle_id"`
	EvaluatedAt time.Time   `json:"evaluated_at"`
	Facts       []InputFact `json:"facts"`
}

func (i EvaluationInput) Validate() error {
	if strings.TrimSpace(i.NodeID) == "" {
		return errors.New("rule evaluation node_id is required")
	}
	if i.EvaluatedAt.IsZero() {
		return errors.New("rule evaluation evaluated_at is required")
	}
	if len(i.Facts) == 0 {
		return errors.New("rule evaluation requires at least one input fact")
	}
	for idx, fact := range i.Facts {
		if err := fact.Validate(); err != nil {
			return fmt.Errorf("rule evaluation fact %d: %w", idx, err)
		}
	}
	return nil
}

type EvaluationResult struct {
	RuleID           string           `json:"rule_id"`
	RuleVersion      string           `json:"rule_version"`
	TargetEntity     string           `json:"target_entity"`
	Decision         string           `json:"decision"`
	SuggestedAction  string           `json:"suggested_action"`
	ExecutionPosture ExecutionPosture `json:"execution_posture"`
	FiredAt          time.Time        `json:"fired_at"`
}

func (r EvaluationResult) normalized(input EvaluationInput) EvaluationResult {
	if r.ExecutionPosture == "" {
		r.ExecutionPosture = ExecutionPostureObserveOnly
	}
	if r.FiredAt.IsZero() {
		r.FiredAt = input.EvaluatedAt
	}
	return r
}

func (r EvaluationResult) Validate() error {
	if strings.TrimSpace(r.RuleID) == "" || strings.TrimSpace(r.Decision) == "" || strings.TrimSpace(r.SuggestedAction) == "" {
		return errors.New("rule id, decision, and suggested action are required")
	}
	if strings.TrimSpace(string(r.ExecutionPosture)) == "" {
		return errors.New("rule execution posture is required")
	}
	if r.FiredAt.IsZero() {
		return errors.New("rule fired_at is required")
	}
	return nil
}

type TracePayload struct {
	ID               string           `json:"entity_id"`
	RuleID           string           `json:"rule_id"`
	RuleVersion      string           `json:"rule_version"`
	NodeID           string           `json:"node_id"`
	VehicleID        string           `json:"vehicle_id,omitempty"`
	TargetEntity     string           `json:"target_entity,omitempty"`
	Decision         string           `json:"decision"`
	SuggestedAction  string           `json:"suggested_action"`
	ExecutionPosture ExecutionPosture `json:"execution_posture"`
	FiredAt          time.Time        `json:"fired_at"`
	InputFacts       []InputFact      `json:"input_facts"`
	InputHash        string           `json:"input_hash"`
}

func NewTracePayload(input EvaluationInput, result EvaluationResult) (*TracePayload, error) {
	if err := input.Validate(); err != nil {
		return nil, err
	}
	result = result.normalized(input)
	if err := result.Validate(); err != nil {
		return nil, err
	}
	inputHash, err := HashInput(input)
	if err != nil {
		return nil, err
	}
	trace := &TracePayload{
		ID:               TraceEntityID(result.RuleID, input.NodeID, result.FiredAt, inputHash),
		RuleID:           result.RuleID,
		RuleVersion:      result.RuleVersion,
		NodeID:           input.NodeID,
		VehicleID:        input.VehicleID,
		TargetEntity:     result.TargetEntity,
		Decision:         result.Decision,
		SuggestedAction:  result.SuggestedAction,
		ExecutionPosture: result.ExecutionPosture,
		FiredAt:          result.FiredAt,
		InputFacts:       append([]InputFact(nil), input.Facts...),
		InputHash:        inputHash,
	}
	if err := trace.Validate(); err != nil {
		return nil, err
	}
	return trace, nil
}

func (p *TracePayload) Schema() message.Type { return TraceType }
func (p *TracePayload) EntityID() string     { return p.ID }
func (p *TracePayload) IndexingProfile() string {
	return vocabulary.IndexingProfileTrace
}

func (p *TracePayload) Validate() error {
	if p == nil {
		return errors.New("rule trace payload is required")
	}
	if strings.TrimSpace(p.ID) == "" || strings.TrimSpace(p.RuleID) == "" || strings.TrimSpace(p.NodeID) == "" {
		return errors.New("rule trace entity id, rule id, and node id are required")
	}
	if strings.TrimSpace(p.Decision) == "" || strings.TrimSpace(p.SuggestedAction) == "" {
		return errors.New("rule trace decision and suggested action are required")
	}
	if strings.TrimSpace(string(p.ExecutionPosture)) == "" {
		return errors.New("rule trace execution posture is required")
	}
	if p.FiredAt.IsZero() {
		return errors.New("rule trace fired_at is required")
	}
	if len(p.InputFacts) == 0 {
		return errors.New("rule trace requires input facts")
	}
	for idx, fact := range p.InputFacts {
		if err := fact.Validate(); err != nil {
			return fmt.Errorf("rule trace input fact %d: %w", idx, err)
		}
	}
	if strings.TrimSpace(p.InputHash) == "" {
		return errors.New("rule trace input_hash is required")
	}
	if _, err := p.InputFactsJSON(); err != nil {
		return err
	}
	return nil
}

func (p *TracePayload) MarshalJSON() ([]byte, error) {
	type alias TracePayload
	return json.Marshal((*alias)(p))
}

func (p *TracePayload) UnmarshalJSON(data []byte) error {
	type alias TracePayload
	return json.Unmarshal(data, (*alias)(p))
}

func (p *TracePayload) Triples() []message.Triple {
	now := nonZeroTime(p.FiredAt)
	inputFacts, _ := p.InputFactsJSON()
	triples := []message.Triple{
		triple(p.ID, PredicateTraceRuleID, p.RuleID, SourceRuleEngine, now, 1),
		triple(p.ID, PredicateTraceRuleVersion, p.RuleVersion, SourceRuleEngine, now, 1),
		triple(p.ID, PredicateTraceNode, p.NodeID, SourceRuleEngine, now, 1),
		triple(p.ID, PredicateTraceDecision, p.Decision, SourceRuleEngine, now, 1),
		triple(p.ID, PredicateTraceSuggestedAction, p.SuggestedAction, SourceRuleEngine, now, 1),
		triple(p.ID, PredicateTraceExecutionPosture, string(p.ExecutionPosture), SourceRuleEngine, now, 1),
		triple(p.ID, PredicateTraceInputCount, len(p.InputFacts), SourceRuleEngine, now, 1),
		triple(p.ID, PredicateTraceInputHash, p.InputHash, SourceRuleEngine, now, 1),
		triple(p.ID, PredicateTraceInputFactsJSON, inputFacts, SourceRuleEngine, now, 1),
		triple(p.ID, PredicateTraceFiredUnixMS, now.UnixMilli(), SourceRuleEngine, now, 1),
	}
	if p.VehicleID != "" {
		triples = append(triples, triple(p.ID, PredicateTraceVehicle, p.VehicleID, SourceRuleEngine, now, 1))
	}
	if p.TargetEntity != "" {
		triples = append(triples, triple(p.ID, PredicateTraceTarget, p.TargetEntity, SourceRuleEngine, now, 1))
	}
	return triples
}

func (p *TracePayload) InputFactsJSON() (string, error) {
	raw, err := json.Marshal(p.InputFacts)
	if err != nil {
		return "", fmt.Errorf("marshal rule trace input facts: %w", err)
	}
	return string(raw), nil
}

func (p *TracePayload) Projection() (graphprojection.Projection, error) {
	if err := p.Validate(); err != nil {
		return graphprojection.Projection{}, err
	}
	return graphprojection.ProjectionFromPayload(p, TraceType, p.FiredAt), nil
}

func HashInput(input EvaluationInput) (string, error) {
	raw, err := json.Marshal(input)
	if err != nil {
		return "", fmt.Errorf("marshal rule evaluation input: %w", err)
	}
	sum := sha256.Sum256(raw)
	return "sha256:" + hex.EncodeToString(sum[:]), nil
}

func TraceEntityID(ruleID, nodeID string, firedAt time.Time, inputHash string) string {
	shortHash := strings.TrimPrefix(inputHash, "sha256:")
	if len(shortHash) > 12 {
		shortHash = shortHash[:12]
	}
	return fmt.Sprintf(
		"c360.semlink.robotics.fleet.rule.trace-%s-%s-%d-%s",
		safeToken(ruleID),
		safeToken(nodeID),
		firedAt.UnixMilli(),
		shortHash,
	)
}

func RegisterPayloads(reg *payloadregistry.Registry) error {
	return reg.Register(&payloadregistry.Registration{
		Domain:      TraceType.Domain,
		Category:    TraceType.Category,
		Version:     TraceType.Version,
		Description: "SemLink local rule trace evidence",
		Factory:     func() any { return &TracePayload{} },
	})
}

func mustType(domain, category, version string) semType {
	return semType{Domain: domain, Category: category, Version: version}
}

func triple(subject, predicate string, object any, source string, timestamp time.Time, confidence float64) message.Triple {
	return graphprojection.Triple(subject, predicate, object, source, timestamp, confidence)
}

func nonZeroTime(t time.Time) time.Time {
	if t.IsZero() {
		return time.Now()
	}
	return t
}

func safeToken(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	var b strings.Builder
	lastDash := false
	for _, r := range s {
		ok := (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9')
		if ok {
			b.WriteRune(r)
			lastDash = false
			continue
		}
		if !lastDash {
			b.WriteByte('-')
			lastDash = true
		}
	}
	out := strings.Trim(b.String(), "-")
	if out == "" {
		return "unknown"
	}
	return out
}

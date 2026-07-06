package commandgate

import (
	"errors"
	"strings"
	"time"
)

var ErrHardwareTransmitBlocked = errors.New("commandgate: hardware transmit blocked")

const (
	HardwareTransmitBlockPolicyID      = "hardware-transmit-blocked-until-openspec"
	HardwareTransmitRequiredChange     = "accepted OpenSpec change defining hardware authorization and safety evidence"
	HardwareTransmitBlockStatus        = "hardware-transmit-blocked"
	HardwareTransmitCompanionMeshScope = "pivot-companion-mesh"
)

type HardwareTransmitBlocker struct {
	PolicyID       string
	RequiredChange string
	Scope          string
	Now            func() time.Time
}

type HardwareTransmitRequest struct {
	RuntimeMode       RuntimeMode `json:"runtime_mode"`
	SafetyProfile     string      `json:"safety_profile"`
	TargetEntity      string      `json:"target_entity"`
	Verb              string      `json:"verb"`
	RequestedBy       string      `json:"requested_by"`
	TargetSystemID    uint8       `json:"target_system_id"`
	TargetComponentID uint8       `json:"target_component_id"`
}

type HardwareTransmitBlockEvidence struct {
	Accepted          bool        `json:"accepted"`
	Status            string      `json:"status"`
	PolicyID          string      `json:"policy_id"`
	RequiredChange    string      `json:"required_change"`
	Scope             string      `json:"scope"`
	RuntimeMode       RuntimeMode `json:"runtime_mode"`
	SafetyProfile     string      `json:"safety_profile"`
	TargetEntity      string      `json:"target_entity"`
	Verb              string      `json:"verb"`
	RequestedBy       string      `json:"requested_by"`
	TargetSystemID    uint8       `json:"target_system_id"`
	TargetComponentID uint8       `json:"target_component_id"`
	ObservedAt        time.Time   `json:"observed_at"`
	Reason            string      `json:"reason"`
}

func DefaultHardwareTransmitBlocker() HardwareTransmitBlocker {
	return HardwareTransmitBlocker{
		PolicyID:       HardwareTransmitBlockPolicyID,
		RequiredChange: HardwareTransmitRequiredChange,
		Scope:          HardwareTransmitCompanionMeshScope,
	}
}

func (b HardwareTransmitBlocker) Check(req HardwareTransmitRequest) (HardwareTransmitBlockEvidence, error) {
	b = b.withDefaults()
	req.RuntimeMode = RuntimeMode(normalize(string(req.RuntimeMode)))
	req.SafetyProfile = normalize(req.SafetyProfile)
	req.TargetEntity = strings.TrimSpace(req.TargetEntity)
	req.Verb = normalize(req.Verb)
	req.RequestedBy = strings.TrimSpace(req.RequestedBy)
	return HardwareTransmitBlockEvidence{
		Accepted:          false,
		Status:            HardwareTransmitBlockStatus,
		PolicyID:          b.PolicyID,
		RequiredChange:    b.RequiredChange,
		Scope:             b.Scope,
		RuntimeMode:       req.RuntimeMode,
		SafetyProfile:     req.SafetyProfile,
		TargetEntity:      req.TargetEntity,
		Verb:              req.Verb,
		RequestedBy:       req.RequestedBy,
		TargetSystemID:    req.TargetSystemID,
		TargetComponentID: req.TargetComponentID,
		ObservedAt:        b.now(),
		Reason:            "hardware command transmit is out of scope for the current companion-mesh change",
	}, ErrHardwareTransmitBlocked
}

func (b HardwareTransmitBlocker) withDefaults() HardwareTransmitBlocker {
	if b.PolicyID == "" {
		b.PolicyID = HardwareTransmitBlockPolicyID
	}
	if b.RequiredChange == "" {
		b.RequiredChange = HardwareTransmitRequiredChange
	}
	if b.Scope == "" {
		b.Scope = HardwareTransmitCompanionMeshScope
	}
	return b
}

func (b HardwareTransmitBlocker) now() time.Time {
	if b.Now != nil {
		return b.Now()
	}
	return time.Now()
}

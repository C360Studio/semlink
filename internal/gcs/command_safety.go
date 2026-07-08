package gcs

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/c360studio/semlink/internal/commandgate"
	"github.com/c360studio/semlink/internal/handoff"
)

const (
	handoffCommandRequestedBy = "semlink-handoff-api"
	defaultAutopilotComponent = 1
)

func (s *Server) rejectHandoffHardwareCommand(w http.ResponseWriter, targetEntity, verb string) bool {
	if s == nil || s.handoffProfile == nil ||
		s.handoffProfile.CommandRuntimeMode != handoff.CommandRuntimeHardwareReadonly {
		return false
	}

	targetSystemID := s.targetSystemIDForCommand(targetEntity)
	var targetComponentID uint8
	if targetSystemID != 0 {
		targetComponentID = defaultAutopilotComponent
	}
	blocker := commandgate.DefaultHardwareTransmitBlocker()
	blocker.Scope = commandgate.HardwareTransmitHandoffScope
	block, _ := blocker.Check(commandgate.HardwareTransmitRequest{
		RuntimeMode:       commandgate.RuntimeModeHardware,
		SafetyProfile:     string(s.handoffProfile.CommandRuntimeMode),
		TargetEntity:      targetEntity,
		Verb:              verb,
		RequestedBy:       handoffCommandRequestedBy,
		TargetSystemID:    targetSystemID,
		TargetComponentID: targetComponentID,
	})
	result := commandgate.TransmitResult{
		Status:        block.Status,
		StartedAt:     block.ObservedAt,
		HardwareBlock: &block,
	}
	s.store.RecordCommandGateResult(result)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusConflict)
	_ = json.NewEncoder(w).Encode(result)
	return true
}

func (s *Server) targetSystemIDForCommand(targetEntity string) uint8 {
	targetEntity = strings.TrimSpace(targetEntity)
	if targetEntity == "" {
		return 0
	}
	if id, err := systemIDFromEntity(targetEntity); err == nil {
		return id
	}
	if s == nil || s.store == nil {
		return 0
	}
	for _, vehicle := range s.store.Snapshot().Vehicles {
		if vehicle.EntityID == targetEntity {
			return vehicle.SystemID
		}
	}
	return 0
}

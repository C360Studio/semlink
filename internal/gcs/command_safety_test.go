package gcs

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/c360studio/semlink/internal/commandgate"
	"github.com/c360studio/semlink/internal/handoff"
	"github.com/c360studio/semlink/internal/mavlink"
	"github.com/c360studio/semlink/internal/projector"
)

func TestHandoffCommandsRejectHardwareTransmitAndRecordEvidence(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	targetEntity := "c360.semlink.robotics.fleet.drone.uav-042"
	store := NewStore("nats://demo", true)
	store.ApplyProjectorSnapshot([]projector.VehicleStatePayload{{
		ID:               targetEntity,
		Callsign:         "BOAT-042",
		SystemID:         42,
		VehicleType:      "surface-boat",
		Mode:             "manual",
		FlightStatus:     "active",
		LinkStatus:       "online",
		BatteryRemaining: 80,
		LastSeen:         now,
	}}, nil)
	profile := handoff.DefaultProfile()
	profile.CommandRuntimeMode = handoff.CommandRuntimeHardwareReadonly
	profile.HardwareTransmitEnabled = false
	server := NewServer(store, nil, "", ServerOptions{
		HandoffProfile: &profile,
	})

	body := bytes.NewBufferString(`{"vehicle_id":"` + targetEntity + `","verb":"request-autopilot-version"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/commands", body)
	rr := httptest.NewRecorder()
	server.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusConflict {
		t.Fatalf("status = %d body = %s", rr.Code, rr.Body.String())
	}
	var result commandgate.TransmitResult
	if err := json.Unmarshal(rr.Body.Bytes(), &result); err != nil {
		t.Fatalf("decode command response: %v", err)
	}
	if result.Accepted || result.Status != commandgate.HardwareTransmitBlockStatus {
		t.Fatalf("result = %#v", result)
	}
	if result.HardwareBlock == nil {
		t.Fatal("hardware block missing from command response")
	}
	assertHandoffHardwareBlock(t, *result.HardwareBlock, targetEntity)

	snapshot := store.Snapshot()
	if len(snapshot.Commands) != 0 {
		t.Fatalf("commands = %#v, want no command intent writes", snapshot.Commands)
	}
	if len(snapshot.CommandGates) != 1 {
		t.Fatalf("command gates = %#v, want one hardware block", snapshot.CommandGates)
	}
	if snapshot.CommandGates[0].HardwareBlock == nil {
		t.Fatalf("command gate missing hardware block: %#v", snapshot.CommandGates[0])
	}
	assertHandoffHardwareBlock(t, *snapshot.CommandGates[0].HardwareBlock, targetEntity)

	evidence := server.EvidenceBundle(now)
	if len(evidence.Commands) != 1 {
		t.Fatalf("evidence commands = %#v, want one command-gate entry", evidence.Commands)
	}
	if evidence.Commands[0].Kind != "command-gate" || evidence.Commands[0].Gate == nil ||
		evidence.Commands[0].Gate.HardwareBlock == nil {
		t.Fatalf("evidence command gate = %#v", evidence.Commands[0])
	}
	assertHandoffHardwareBlock(t, *evidence.Commands[0].Gate.HardwareBlock, targetEntity)
	if evidence.Profile.Command.RuntimeMode != string(handoff.CommandRuntimeHardwareReadonly) ||
		evidence.Profile.Command.HardwareTransmitEnabled ||
		evidence.Profile.Command.HardwareTransmitStatus != "blocked" {
		t.Fatalf("profile command evidence = %#v", evidence.Profile.Command)
	}
}

func TestEvidencePreservesSimulatorCommandGateWithoutHardwareAuthorization(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	targetEntity := "c360.semlink.robotics.fleet.boat.boat-042"
	store := NewStore("nats://demo", true)
	profile := handoff.DefaultProfile()
	profile.CommandRuntimeMode = handoff.CommandRuntimeSimulator
	profile.HardwareTransmitEnabled = false

	store.RecordCommandGateResult(commandgate.TransmitResult{
		Accepted:  true,
		Status:    "accepted",
		StartedAt: now,
		Preflight: commandgate.PreflightResult{
			Accepted: true,
			Evidence: commandgate.PreflightEvidence{
				RuntimeMode:              commandgate.RuntimeModeSimulator,
				SafetyProfile:            commandgate.DefaultSafetyProfile,
				TargetEntity:             targetEntity,
				Verb:                     commandgate.VerbRequestAutopilotVersion,
				RequestedBy:              "simulator-harness",
				SenderSystemID:           250,
				SenderComponentID:        191,
				Attempts:                 1,
				MaxAttempts:              3,
				LocalOverride:            true,
				ACKRequired:              true,
				PostStatePollingRequired: true,
				SimulatorConfirmed:       true,
				AbortReady:               true,
			},
		},
		Frames: []commandgate.FrameEvidence{{
			Attempt:            1,
			Sequence:           12,
			MessageID:          mavlink.MessageCommandLong,
			CommandID:          mavlink.MAVCmdRequestMessage,
			RequestedMessageID: mavlink.MessageAutopilotVersion,
			SenderSystemID:     250,
			SenderComponentID:  191,
			TargetSystemID:     42,
			TargetComponentID:  defaultAutopilotComponent,
			FrameBytes:         42,
		}},
		ACKs: []commandgate.ACKEvidence{{
			Attempt:           1,
			CommandID:         mavlink.MAVCmdRequestMessage,
			Result:            mavlink.MAVResultAccepted,
			SourceSystemID:    42,
			SourceComponentID: defaultAutopilotComponent,
			ObservedAt:        now,
		}},
		PostState: commandgate.PostStateEvidence{
			Observed:     true,
			TargetEntity: targetEntity,
			Summary:      "AUTOPILOT_VERSION observed",
			ObservedAt:   now,
		},
	})

	server := NewServer(store, nil, "", ServerOptions{
		HandoffProfile: &profile,
	})
	evidence := server.EvidenceBundle(now)
	if len(evidence.Commands) != 1 {
		t.Fatalf("evidence commands = %#v, want one command gate", evidence.Commands)
	}
	gate := evidence.Commands[0].Gate
	if evidence.Commands[0].Kind != "command-gate" || gate == nil {
		t.Fatalf("evidence command = %#v, want command gate", evidence.Commands[0])
	}
	if !gate.PreflightAccepted ||
		!gate.SimulatorOnly ||
		gate.FrameCount != 1 ||
		gate.ACKCount != 1 ||
		!gate.ACKAccepted ||
		!gate.PostStateObserved ||
		gate.HardwareTransmitAuthorized ||
		gate.HardwareBlock != nil {
		t.Fatalf("simulator command gate = %#v", gate)
	}
	if evidence.Profile.Command.RuntimeMode != string(handoff.CommandRuntimeSimulator) ||
		evidence.Profile.Command.HardwareTransmitEnabled ||
		evidence.Profile.Command.HardwareTransmitStatus != "blocked" {
		t.Fatalf("profile command evidence = %#v", evidence.Profile.Command)
	}
}

func assertHandoffHardwareBlock(t *testing.T, block commandgate.HardwareTransmitBlockEvidence, targetEntity string) {
	t.Helper()
	if block.Accepted ||
		block.Status != commandgate.HardwareTransmitBlockStatus ||
		block.Scope != commandgate.HardwareTransmitHandoffScope ||
		block.RuntimeMode != commandgate.RuntimeModeHardware ||
		block.SafetyProfile != string(handoff.CommandRuntimeHardwareReadonly) ||
		block.TargetEntity != targetEntity ||
		block.Verb != commandgate.VerbRequestAutopilotVersion ||
		block.RequestedBy != handoffCommandRequestedBy ||
		block.TargetSystemID != 42 ||
		block.TargetComponentID != defaultAutopilotComponent ||
		block.RequiredChange == "" ||
		block.Reason == "" {
		t.Fatalf("hardware block = %#v", block)
	}
}

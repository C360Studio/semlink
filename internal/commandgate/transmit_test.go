package commandgate

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/c360studio/semlink/internal/mavlink"
)

func TestSimulatorTransmitGateSendsCommandLongAndRequiresEvidence(t *testing.T) {
	ctx := context.Background()
	tx := &recordingTransmitter{}
	acks := &scriptedACKObserver{acks: []ACKEvidence{{
		CommandID:         mavlink.MAVCmdRequestMessage,
		Result:            mavlink.MAVResultAccepted,
		SourceSystemID:    42,
		SourceComponentID: 1,
		ObservedAt:        time.Unix(170, 0),
	}}}
	poller := &scriptedPostStatePoller{state: PostStateEvidence{
		Observed:     true,
		TargetEntity: "c360.semlink.robotics.fleet.drone.uav-001",
		Summary:      "autopilot-version observed",
		ObservedAt:   time.Unix(171, 0),
	}}
	gate := SimulatorTransmitGate{
		Transmitter:     tx,
		ACKObserver:     acks,
		PostStatePoller: poller,
		Now:             func() time.Time { return time.Unix(169, 0) },
	}

	result, err := gate.Transmit(ctx, validSimulatorTransmit())
	if err != nil {
		t.Fatalf("Transmit: %v", err)
	}
	if !result.Accepted {
		t.Fatalf("Accepted = false, status = %q", result.Status)
	}
	if len(tx.frames) != 1 {
		t.Fatalf("transmitted frames = %d, want 1", len(tx.frames))
	}
	frame, err := mavlink.DecodeFrame(tx.frames[0])
	if err != nil {
		t.Fatalf("DecodeFrame: %v", err)
	}
	if frame.SystemID != 250 || frame.ComponentID != 191 {
		t.Fatalf("sender = %d/%d, want 250/191", frame.SystemID, frame.ComponentID)
	}
	if frame.MessageID != mavlink.MessageCommandLong {
		t.Fatalf("message id = %d, want COMMAND_LONG", frame.MessageID)
	}
	command, err := mavlink.DecodeCommandLongPayload(frame.Payload)
	if err != nil {
		t.Fatalf("DecodeCommandLongPayload: %v", err)
	}
	if command.TargetSystem != 1 || command.TargetComponent != 1 {
		t.Fatalf("target = %d/%d, want 1/1", command.TargetSystem, command.TargetComponent)
	}
	if command.Command != mavlink.MAVCmdRequestMessage {
		t.Fatalf("command = %d, want request-message", command.Command)
	}
	if command.Params[0] != float32(mavlink.MessageAutopilotVersion) {
		t.Fatalf("requested message param = %f, want AUTOPILOT_VERSION", command.Params[0])
	}
	if len(result.ACKs) != 1 || !result.ACKs[0].Accepted() {
		t.Fatalf("acks = %#v, want accepted ACK evidence", result.ACKs)
	}
	if !result.PostState.Observed {
		t.Fatalf("post state = %#v, want observed", result.PostState)
	}
}

func TestSimulatorTransmitGateRetriesUntilAcceptedACK(t *testing.T) {
	tx := &recordingTransmitter{}
	acks := &scriptedACKObserver{acks: []ACKEvidence{
		{CommandID: mavlink.MAVCmdRequestMessage, Result: mavlink.MAVResultTemporarilyRejected},
		{CommandID: mavlink.MAVCmdRequestMessage, Result: mavlink.MAVResultAccepted},
	}}
	gate := SimulatorTransmitGate{
		Transmitter:     tx,
		ACKObserver:     acks,
		PostStatePoller: &scriptedPostStatePoller{state: PostStateEvidence{Observed: true}},
	}
	req := validSimulatorTransmit()
	req.Preflight.Attempts = 2

	result, err := gate.Transmit(context.Background(), req)
	if err != nil {
		t.Fatalf("Transmit: %v", err)
	}
	if !result.Accepted {
		t.Fatalf("Accepted = false, status = %q", result.Status)
	}
	if len(tx.frames) != 2 {
		t.Fatalf("transmitted frames = %d, want 2", len(tx.frames))
	}
	second, err := mavlink.DecodeFrame(tx.frames[1])
	if err != nil {
		t.Fatalf("DecodeFrame: %v", err)
	}
	command, err := mavlink.DecodeCommandLongPayload(second.Payload)
	if err != nil {
		t.Fatalf("DecodeCommandLongPayload: %v", err)
	}
	if command.Confirmation != 1 {
		t.Fatalf("second confirmation = %d, want retry confirmation 1", command.Confirmation)
	}
}

func TestSimulatorTransmitGateRejectsFailedPreflightWithoutTransmit(t *testing.T) {
	tx := &recordingTransmitter{}
	gate := SimulatorTransmitGate{
		Transmitter:     tx,
		ACKObserver:     &scriptedACKObserver{},
		PostStatePoller: &scriptedPostStatePoller{},
	}
	req := validSimulatorTransmit()
	req.Preflight.RuntimeMode = RuntimeModeHardware

	result, err := gate.Transmit(context.Background(), req)
	if !errors.Is(err, ErrPreflightRejected) {
		t.Fatalf("error = %v, want ErrPreflightRejected", err)
	}
	if result.Accepted {
		t.Fatal("Accepted = true, want false")
	}
	if len(tx.frames) != 0 {
		t.Fatalf("transmitted frames = %d, want 0", len(tx.frames))
	}
}

func TestSimulatorTransmitGateRejectsMissingPostStateAfterACK(t *testing.T) {
	tx := &recordingTransmitter{}
	gate := SimulatorTransmitGate{
		Transmitter: tx,
		ACKObserver: &scriptedACKObserver{acks: []ACKEvidence{{
			CommandID: mavlink.MAVCmdRequestMessage,
			Result:    mavlink.MAVResultAccepted,
		}}},
		PostStatePoller: &scriptedPostStatePoller{state: PostStateEvidence{Observed: false}},
	}

	result, err := gate.Transmit(context.Background(), validSimulatorTransmit())
	if !errors.Is(err, ErrPostStateNotObserved) {
		t.Fatalf("error = %v, want ErrPostStateNotObserved", err)
	}
	if result.Accepted {
		t.Fatal("Accepted = true, want false")
	}
	if len(result.ACKs) != 1 || !result.ACKs[0].Accepted() {
		t.Fatalf("acks = %#v, want accepted ACK before post-state failure", result.ACKs)
	}
}

func validSimulatorTransmit() TransmitRequest {
	return TransmitRequest{
		Preflight:          validSimulatorPreflight(),
		Sequence:           10,
		TargetSystemID:     1,
		TargetComponentID:  1,
		RequestedMessageID: mavlink.MessageAutopilotVersion,
	}
}

type recordingTransmitter struct {
	frames [][]byte
}

func (r *recordingTransmitter) TransmitMAVLink(_ context.Context, frame []byte) error {
	copied := append([]byte(nil), frame...)
	r.frames = append(r.frames, copied)
	return nil
}

type scriptedACKObserver struct {
	acks  []ACKEvidence
	calls int
}

func (s *scriptedACKObserver) AwaitCommandACK(_ context.Context, expected ACKExpectation) (ACKEvidence, error) {
	s.calls++
	if len(s.acks) == 0 {
		return ACKEvidence{CommandID: expected.CommandID, Result: mavlink.MAVResultFailed}, nil
	}
	ack := s.acks[0]
	s.acks = s.acks[1:]
	ack.Attempt = expected.Attempt
	return ack, nil
}

type scriptedPostStatePoller struct {
	state PostStateEvidence
	calls int
}

func (s *scriptedPostStatePoller) PollPostState(_ context.Context, expected PostStateExpectation) (PostStateEvidence, error) {
	s.calls++
	if s.state.TargetEntity == "" {
		s.state.TargetEntity = expected.TargetEntity
	}
	return s.state, nil
}

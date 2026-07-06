package mavlink

import "testing"

func TestCommandLongRequestMessagePayload(t *testing.T) {
	payload := RequestMessageCommandPayload(42, 1, MessageAutopilotVersion, 2)
	got, err := DecodeCommandLongPayload(payload)
	if err != nil {
		t.Fatalf("DecodeCommandLongPayload: %v", err)
	}

	if got.TargetSystem != 42 || got.TargetComponent != 1 {
		t.Fatalf("target = %d/%d, want 42/1", got.TargetSystem, got.TargetComponent)
	}
	if got.Command != MAVCmdRequestMessage {
		t.Fatalf("command = %d, want %d", got.Command, MAVCmdRequestMessage)
	}
	if got.Confirmation != 2 {
		t.Fatalf("confirmation = %d, want 2", got.Confirmation)
	}
	if got.Params[0] != float32(MessageAutopilotVersion) {
		t.Fatalf("param1 = %f, want AUTOPILOT_VERSION id", got.Params[0])
	}
}

func TestDecodeCommandAck(t *testing.T) {
	payload := CommandAckPayload(MAVCmdRequestMessage, MAVResultAccepted, 100, 0, 250, 191)
	frame, err := EncodeV2(9, 42, 1, MessageCommandAck, payload)
	if err != nil {
		t.Fatalf("EncodeV2: %v", err)
	}

	msg, err := DecodeMessage(frame)
	if err != nil {
		t.Fatalf("DecodeMessage: %v", err)
	}
	got, ok := msg.(CommandAck)
	if !ok {
		t.Fatalf("message type = %T, want CommandAck", msg)
	}
	if got.Command != MAVCmdRequestMessage || got.Result != MAVResultAccepted {
		t.Fatalf("ack command/result = %d/%d", got.Command, got.Result)
	}
	if got.TargetSystem != 250 || got.TargetComponent != 191 {
		t.Fatalf("ack target = %d/%d, want 250/191", got.TargetSystem, got.TargetComponent)
	}
	if !got.Accepted() {
		t.Fatal("Accepted = false, want true")
	}
}

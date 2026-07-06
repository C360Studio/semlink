package commandgate

import (
	"errors"
	"testing"
	"time"
)

func TestHardwareTransmitBlockerRequiresSeparateOpenSpecChange(t *testing.T) {
	blocker := DefaultHardwareTransmitBlocker()
	blocker.Now = func() time.Time { return time.Unix(200, 0) }

	result, err := blocker.Check(HardwareTransmitRequest{
		RuntimeMode:       " HARDWARE ",
		SafetyProfile:     " HARDWARE_TRANSMIT_V1 ",
		TargetEntity:      " c360.semlink.robotics.fleet.drone.uav-001 ",
		Verb:              " HOLD_POSITION ",
		RequestedBy:       "operator:alpha",
		TargetSystemID:    1,
		TargetComponentID: 1,
	})
	if !errors.Is(err, ErrHardwareTransmitBlocked) {
		t.Fatalf("error = %v, want ErrHardwareTransmitBlocked", err)
	}
	if result.Accepted {
		t.Fatal("Accepted = true, want false")
	}
	if result.Status != HardwareTransmitBlockStatus {
		t.Fatalf("status = %q, want %q", result.Status, HardwareTransmitBlockStatus)
	}
	if result.PolicyID != HardwareTransmitBlockPolicyID {
		t.Fatalf("policy id = %q, want %q", result.PolicyID, HardwareTransmitBlockPolicyID)
	}
	if result.RequiredChange == "" {
		t.Fatal("required change is empty")
	}
	if result.RuntimeMode != RuntimeModeHardware {
		t.Fatalf("runtime mode = %q, want hardware", result.RuntimeMode)
	}
	if result.SafetyProfile != "hardware-transmit-v1" {
		t.Fatalf("safety profile = %q, want normalized profile", result.SafetyProfile)
	}
	if result.Verb != "hold-position" {
		t.Fatalf("verb = %q, want normalized verb", result.Verb)
	}
	if result.TargetSystemID != 1 || result.TargetComponentID != 1 {
		t.Fatalf("target = %d/%d, want 1/1", result.TargetSystemID, result.TargetComponentID)
	}
	if !result.ObservedAt.Equal(time.Unix(200, 0)) {
		t.Fatalf("observed at = %v", result.ObservedAt)
	}
}

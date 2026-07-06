package commandgate

import "testing"

func TestSimulatorPreflightAcceptsOnlyWhenAllSafetyChecksPass(t *testing.T) {
	req := validSimulatorPreflight()
	req.RuntimeMode = " SIMULATOR "
	req.SafetyProfile = " SIMULATOR_READBACK_V1 "
	req.Verb = " REQUEST_AUTOPILOT_VERSION "

	result := DefaultSimulatorPreflightGate().Check(req)
	if !result.Accepted {
		t.Fatalf("Accepted = false, checks = %#v", result.Checks)
	}
	assertCheck(t, result, "simulator-only", true)
	assertCheck(t, result, "safety-profile", true)
	assertCheck(t, result, "narrow-allowlist", true)
	assertCheck(t, result, "ack-required", true)
	assertCheck(t, result, "post-state-polling", true)
	assertCheck(t, result, "abort-ready", true)
	if result.Evidence.Verb != VerbRequestAutopilotVersion {
		t.Fatalf("evidence verb = %q", result.Evidence.Verb)
	}
	if result.Evidence.MaxAttempts != 3 {
		t.Fatalf("evidence max attempts = %d, want 3", result.Evidence.MaxAttempts)
	}
}

func TestSimulatorPreflightRejectsUnknownSafetyProfile(t *testing.T) {
	req := validSimulatorPreflight()
	req.SafetyProfile = "hardware-transmit-v1"

	result := DefaultSimulatorPreflightGate().Check(req)
	if result.Accepted {
		t.Fatal("Accepted = true, want false for non-allowlisted safety profile")
	}
	assertCheck(t, result, "safety-profile", false)
}

func TestSimulatorPreflightRejectsHardwareMode(t *testing.T) {
	req := validSimulatorPreflight()
	req.RuntimeMode = RuntimeModeHardware

	result := DefaultSimulatorPreflightGate().Check(req)
	if result.Accepted {
		t.Fatal("Accepted = true, want false for hardware mode")
	}
	assertCheck(t, result, "simulator-only", false)
}

func TestSimulatorPreflightRejectsVehicleActionOutsideAllowlist(t *testing.T) {
	req := validSimulatorPreflight()
	req.Verb = "hold-position"

	result := DefaultSimulatorPreflightGate().Check(req)
	if result.Accepted {
		t.Fatal("Accepted = true, want false for non-allowlisted vehicle action")
	}
	assertCheck(t, result, "narrow-allowlist", false)
}

func TestSimulatorPreflightRejectsMissingSafetyConfirmations(t *testing.T) {
	req := validSimulatorPreflight()
	req.LocalOverride = false
	req.ACKRequired = false
	req.PostStatePollingRequired = false
	req.SimulatorConfirmed = false
	req.AbortReady = false
	req.Attempts = 4

	result := DefaultSimulatorPreflightGate().Check(req)
	if result.Accepted {
		t.Fatal("Accepted = true, want false for missing safety confirmations")
	}
	assertCheck(t, result, "bounded-attempts", false)
	assertCheck(t, result, "local-override", false)
	assertCheck(t, result, "ack-required", false)
	assertCheck(t, result, "post-state-polling", false)
	assertCheck(t, result, "simulator-confirmed", false)
	assertCheck(t, result, "abort-ready", false)
}

func validSimulatorPreflight() PreflightRequest {
	return PreflightRequest{
		RuntimeMode:              RuntimeModeSimulator,
		SafetyProfile:            DefaultSafetyProfile,
		TargetEntity:             "c360.semlink.robotics.fleet.drone.uav-001",
		Verb:                     VerbRequestAutopilotVersion,
		RequestedBy:              "operator:alpha",
		SenderSystemID:           250,
		SenderComponentID:        191,
		Attempts:                 1,
		LocalOverride:            true,
		ACKRequired:              true,
		PostStatePollingRequired: true,
		SimulatorConfirmed:       true,
		AbortReady:               true,
	}
}

func assertCheck(t *testing.T, result PreflightResult, name string, want bool) {
	t.Helper()
	for _, check := range result.Checks {
		if check.Name == name {
			if check.Passed != want {
				t.Fatalf("check %q passed = %v, want %v", name, check.Passed, want)
			}
			return
		}
	}
	t.Fatalf("missing check %q in %#v", name, result.Checks)
}

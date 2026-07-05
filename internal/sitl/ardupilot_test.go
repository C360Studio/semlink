package sitl

import (
	"strings"
	"testing"
)

func TestArduRoverLaneBuildsHeadlessNoGazeboCommand(t *testing.T) {
	cfg := DefaultArduRoverLane()
	cfg.Frame = FrameSailboatMotor
	cfg.OutputHost = "127.0.0.1"
	cfg.OutputPort = 14550

	args, err := cfg.SimVehicleArgs()
	if err != nil {
		t.Fatalf("SimVehicleArgs() error = %v", err)
	}
	joined := strings.Join(args, " ")

	for _, want := range []string{
		"-v Rover",
		"-f sailboat-motor",
		"--no-mavproxy",
		"--aircraft semlink-ardurover",
		"-A --serial0=udpclient:127.0.0.1:14550",
	} {
		if !strings.Contains(joined, want) {
			t.Fatalf("args %q missing %q", joined, want)
		}
	}
	if strings.Contains(strings.ToLower(joined), "gazebo") {
		t.Fatalf("args must not include Gazebo: %q", joined)
	}
	if strings.Contains(joined, "--map") {
		t.Fatalf("args must not request a graphical map: %q", joined)
	}
}

func TestArduRoverLaneRejectsNonRoverVehicle(t *testing.T) {
	cfg := DefaultArduRoverLane()
	cfg.Vehicle = "Copter"
	if _, err := cfg.SimVehicleArgs(); err == nil {
		t.Fatal("SimVehicleArgs() error = nil, want non-Rover rejection")
	}
}

func TestArduRoverLaneEvidenceSeparatesSITLFromGazeboAndHardware(t *testing.T) {
	cfg := DefaultArduRoverLane()
	evidence, err := cfg.Evidence()
	if err != nil {
		t.Fatalf("Evidence() error = %v", err)
	}

	if evidence.Lane != "ardupilot-sitl" {
		t.Fatalf("Lane = %q", evidence.Lane)
	}
	if evidence.Gazebo {
		t.Fatal("Gazebo = true, want false")
	}
	if evidence.Hardware {
		t.Fatal("Hardware = true, want false")
	}
	if evidence.MAVLinkOutput != "udpclient:127.0.0.1:14550" {
		t.Fatalf("MAVLinkOutput = %q", evidence.MAVLinkOutput)
	}
}

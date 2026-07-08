package sitl

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/c360studio/semlink/internal/handoff"
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

func TestArduRoverLaneUsesHandoffProfileMAVLinkTarget(t *testing.T) {
	profile := handoff.DefaultProfile()
	profile.MAVLinkUDPHost = "boat-alpha.local"
	profile.MAVLinkUDPPort = 14600

	cfg := ArduRoverLaneFromHandoffProfile(profile)
	args, err := cfg.SimVehicleArgs()
	if err != nil {
		t.Fatalf("SimVehicleArgs() error = %v", err)
	}
	joined := strings.Join(args, " ")
	if !strings.Contains(joined, "-A --serial0=udpclient:boat-alpha.local:14600") {
		t.Fatalf("args %q missing handoff profile UDP target", joined)
	}

	evidence, err := cfg.Evidence()
	if err != nil {
		t.Fatalf("Evidence() error = %v", err)
	}
	if evidence.MAVLinkOutput != "udpclient:boat-alpha.local:14600" {
		t.Fatalf("MAVLinkOutput = %q", evidence.MAVLinkOutput)
	}
}

func TestArduRoverScriptsLoadHandoffProfile(t *testing.T) {
	cases := []struct {
		name  string
		path  string
		wants []string
	}{
		{
			name: "local sim_vehicle lane",
			path: filepath.Join("scripts", "ardurover-sitl-lane.sh"),
			wants: []string{
				"SEMLINK_HANDOFF_PROFILE_FILE",
				"configs/handoff/companion.env.example",
				". \"$SEMLINK_HANDOFF_PROFILE_FILE\"",
				"SEMLINK_MAVLINK_UDP_HOST",
				"SEMLINK_MAVLINK_UDP_PORT",
				"--serial0=udpclient:${SEMLINK_MAVLINK_UDP_HOST}:${SEMLINK_MAVLINK_UDP_PORT}",
			},
		},
		{
			name: "docker compose lane",
			path: filepath.Join("scripts", "ardurover-sitl-compose-up.sh"),
			wants: []string{
				"SEMLINK_HANDOFF_PROFILE_FILE",
				"configs/handoff/companion.env.example",
				". \"$SEMLINK_HANDOFF_PROFILE_FILE\"",
				"export SEMLINK_HANDOFF_PROFILE_FILE",
				"export SEMLINK_MAVLINK_UDP_PORT=\"${SEMLINK_MAVLINK_UDP_PORT:-14550}\"",
				"export SEMLINK_MAVLINK_UDP_LISTEN=\"${SEMLINK_MAVLINK_UDP_LISTEN:-:${SEMLINK_MAVLINK_UDP_PORT}}\"",
				"export SEMLINK_MAVLINK_UDP_CONTAINER_PORT=\"${SEMLINK_MAVLINK_UDP_CONTAINER_PORT:-$SEMLINK_MAVLINK_UDP_PORT}\"",
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			body := mustReadRepoFile(t, tc.path)
			for _, want := range tc.wants {
				if !strings.Contains(body, want) {
					t.Fatalf("%s missing %q", tc.path, want)
				}
			}
			if strings.Contains(strings.ToLower(body), "gazebo") {
				t.Fatalf("%s must not require Gazebo", tc.path)
			}
		})
	}
}

func TestSITLComposeUsesSemLinkBuildContext(t *testing.T) {
	body := mustReadRepoFile(t, filepath.Join("compose.sitl.yml"))
	for _, want := range []string{
		"context: ${SEMLINK_ROOT:-.}",
		"dockerfile: docker/ardupilot-sitl/Dockerfile",
		"--serial0=udpclient:semlink:${SEMLINK_MAVLINK_UDP_CONTAINER_PORT:-14550}",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("compose.sitl.yml missing %q", want)
		}
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

func mustReadRepoFile(t *testing.T, path string) string {
	t.Helper()
	body, err := os.ReadFile(filepath.Join("..", "..", path))
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(body)
}

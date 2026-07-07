package handoff

import (
	"bufio"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestParseEnvAcceptsCompanionExample(t *testing.T) {
	env := readExampleEnv(t)

	profile, err := ParseEnv(env)
	if err != nil {
		t.Fatalf("ParseEnv() error = %v", err)
	}

	if profile.NodeID != "boat-alpha" {
		t.Fatalf("NodeID = %q, want boat-alpha", profile.NodeID)
	}
	if profile.VehicleID != "boat-alpha" {
		t.Fatalf("VehicleID = %q, want boat-alpha", profile.VehicleID)
	}
	if profile.Callsign != "BOAT-ALPHA" {
		t.Fatalf("Callsign = %q, want BOAT-ALPHA", profile.Callsign)
	}
	if profile.HTTPListen != ":80" {
		t.Fatalf("HTTPListen = %q, want :80", profile.HTTPListen)
	}
	if profile.BlueOSHostPort != 8081 {
		t.Fatalf("BlueOSHostPort = %d, want 8081", profile.BlueOSHostPort)
	}
	if !profile.EmbeddedNATS {
		t.Fatal("EmbeddedNATS = false, want true")
	}
	if profile.NATSURL != "nats://127.0.0.1:4222" {
		t.Fatalf("NATSURL = %q", profile.NATSURL)
	}
	if profile.MAVLinkUDPListen != ":14550" {
		t.Fatalf("MAVLinkUDPListen = %q, want :14550", profile.MAVLinkUDPListen)
	}
	if profile.MAVLinkUDPHost != "127.0.0.1" || profile.MAVLinkUDPPort != 14550 {
		t.Fatalf("MAVLink UDP target = %s:%d, want 127.0.0.1:14550", profile.MAVLinkUDPHost, profile.MAVLinkUDPPort)
	}
	if profile.Vehicles != 1 || profile.Hz != 5 || profile.Buffer != 10000 {
		t.Fatalf("sim fallback = vehicles:%d hz:%d buffer:%d", profile.Vehicles, profile.Hz, profile.Buffer)
	}
	if len(profile.MeshPeers) != 0 {
		t.Fatalf("MeshPeers len = %d, want 0", len(profile.MeshPeers))
	}
	if profile.CSAPIURL != "" {
		t.Fatalf("CSAPIURL = %q, want empty", profile.CSAPIURL)
	}
	if profile.CSAPIInterval != 2*time.Second {
		t.Fatalf("CSAPIInterval = %s, want 2s", profile.CSAPIInterval)
	}
	if profile.CSAPIObservationInterval != 5*time.Second {
		t.Fatalf("CSAPIObservationInterval = %s, want 5s", profile.CSAPIObservationInterval)
	}
	if profile.CommandRuntimeMode != CommandRuntimeHardwareReadonly {
		t.Fatalf("CommandRuntimeMode = %q, want hardware-readonly", profile.CommandRuntimeMode)
	}
	if profile.HardwareTransmitEnabled {
		t.Fatal("HardwareTransmitEnabled = true, want false")
	}
	if profile.TAKEnabled {
		t.Fatal("TAKEnabled = true, want false")
	}
	if profile.TAKMulticastAddr != "239.2.3.1:6969" || profile.TAKInterval != time.Second {
		t.Fatalf("TAK defaults = %q %s", profile.TAKMulticastAddr, profile.TAKInterval)
	}
}

func TestParseEnvSplitsMeshPeers(t *testing.T) {
	env := readExampleEnv(t)
	env[EnvMeshPeers] = " http://boat-bravo.local:8081 , https://boat-charlie.example "

	profile, err := ParseEnv(env)
	if err != nil {
		t.Fatalf("ParseEnv() error = %v", err)
	}

	if len(profile.MeshPeers) != 2 {
		t.Fatalf("MeshPeers len = %d, want 2", len(profile.MeshPeers))
	}
	if profile.MeshPeers[0] != "http://boat-bravo.local:8081" {
		t.Fatalf("MeshPeers[0] = %q", profile.MeshPeers[0])
	}
	if profile.MeshPeers[1] != "https://boat-charlie.example" {
		t.Fatalf("MeshPeers[1] = %q", profile.MeshPeers[1])
	}
}

func TestParseEnvAllowsSimulatorOnlyMAVLinkInput(t *testing.T) {
	env := readExampleEnv(t)
	env[EnvMAVLinkUDPListen] = ""
	env[EnvCommandRuntimeMode] = string(CommandRuntimeSimulator)

	profile, err := ParseEnv(env)
	if err != nil {
		t.Fatalf("ParseEnv() error = %v", err)
	}

	if profile.MAVLinkUDPListen != "" {
		t.Fatalf("MAVLinkUDPListen = %q, want empty", profile.MAVLinkUDPListen)
	}
	if profile.CommandRuntimeMode != CommandRuntimeSimulator {
		t.Fatalf("CommandRuntimeMode = %q, want simulator", profile.CommandRuntimeMode)
	}
}

func TestParseEnvRejectsHardwareTransmitEnabled(t *testing.T) {
	env := readExampleEnv(t)
	env[EnvHardwareTransmitEnabled] = "true"

	_, err := ParseEnv(env)
	if err == nil {
		t.Fatal("ParseEnv() error = nil, want validation error")
	}
	assertErrorContains(t, err, EnvHardwareTransmitEnabled)
	assertErrorContains(t, err, "must be false until a hardware command OpenSpec change is accepted")
}

func TestParseEnvReportsOperatorFacingErrors(t *testing.T) {
	env := readExampleEnv(t)
	env[EnvNodeID] = " boat alpha "
	env[EnvBlueOSHostPort] = "70000"
	env[EnvMeshPeers] = "boat-bravo"
	env[EnvCSAPIInterval] = "soon"
	env[EnvHardwareTransmitEnabled] = "maybe"

	_, err := ParseEnv(env)
	if err == nil {
		t.Fatal("ParseEnv() error = nil, want validation error")
	}

	var validation *ValidationError
	if !errors.As(err, &validation) {
		t.Fatalf("ParseEnv() error type = %T, want *ValidationError", err)
	}
	if len(validation.Problems) < 5 {
		t.Fatalf("problems len = %d, want at least 5", len(validation.Problems))
	}
	for _, want := range []string{
		"invalid companion handoff profile",
		EnvNodeID,
		EnvBlueOSHostPort,
		EnvMeshPeers,
		EnvCSAPIInterval,
		EnvHardwareTransmitEnabled,
	} {
		assertErrorContains(t, err, want)
	}
}

func TestParseEnvRejectsExternalNATSWithoutURL(t *testing.T) {
	env := readExampleEnv(t)
	env[EnvEmbeddedNATS] = "false"
	env[EnvNATSURL] = ""

	_, err := ParseEnv(env)
	if err == nil {
		t.Fatal("ParseEnv() error = nil, want validation error")
	}
	assertErrorContains(t, err, EnvNATSURL)
	assertErrorContains(t, err, "must be set when SEMLINK_EMBEDDED_NATS is false")
}

func TestMapFromEnviron(t *testing.T) {
	got := MapFromEnviron([]string{
		"SEMLINK_NODE_ID=boat-alpha",
		"SEMLINK_MESH_PEERS=http://boat-bravo.local:8081",
		"ignored",
		"=empty-key",
		"NATS_URL=nats://127.0.0.1:4222",
	})

	if got[EnvNodeID] != "boat-alpha" {
		t.Fatalf("%s = %q", EnvNodeID, got[EnvNodeID])
	}
	if got[EnvMeshPeers] != "http://boat-bravo.local:8081" {
		t.Fatalf("%s = %q", EnvMeshPeers, got[EnvMeshPeers])
	}
	if got[EnvNATSURL] != "nats://127.0.0.1:4222" {
		t.Fatalf("%s = %q", EnvNATSURL, got[EnvNATSURL])
	}
	if _, ok := got[""]; ok {
		t.Fatal("empty key was included")
	}
	if _, ok := got["ignored"]; ok {
		t.Fatal("malformed environment entry was included")
	}
}

func readExampleEnv(t *testing.T) map[string]string {
	t.Helper()

	path := filepath.Join("..", "..", "configs", "handoff", "companion.env.example")
	file, err := os.Open(path)
	if err != nil {
		t.Fatalf("open example env: %v", err)
	}
	defer file.Close()

	env := make(map[string]string)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			t.Fatalf("example env line %q is missing '='", line)
		}
		env[key] = value
	}
	if err := scanner.Err(); err != nil {
		t.Fatalf("scan example env: %v", err)
	}
	return env
}

func assertErrorContains(t *testing.T, err error, want string) {
	t.Helper()
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("error %q does not contain %q", err.Error(), want)
	}
}

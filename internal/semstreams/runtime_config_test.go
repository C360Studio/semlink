package semstreams

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestComposeUsesBeta160PersistentNATSDataWithoutGenerationAttestation(t *testing.T) {
	data, err := os.ReadFile("../../compose.semlink.yml")
	if err != nil {
		t.Fatalf("read compose profile: %v", err)
	}
	compose := string(data)
	if !strings.Contains(compose, "semlink-nats-beta160-data:/data") || !strings.Contains(compose, "volumes:\n  semlink-nats-beta160-data:") {
		t.Fatal("Compose NATS /data is not backed by the beta.160 named volume")
	}
	for _, forbidden := range []string{
		"tmpfs:", "SEMLINK_SEMSTREAMS_STATE_GENERATION",
	} {
		if strings.Contains(compose, forbidden) {
			t.Fatalf("Compose retains forbidden state contract %q", forbidden)
		}
	}
}

func TestRemovedStateCutoverWorkflowHasNoRuntimeOrConfigReferences(t *testing.T) {
	for _, path := range []string{
		"../../cmd/semgcs-demo/main.go",
		"../../configs/handoff/companion.env.example",
		"../../compose.semlink.yml",
		"runtime.go",
	} {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		for _, forbidden := range []string{
			"SEMLINK_SEMSTREAMS_STATE_GENERATION", "check-semstreams-state-cutover", "beta.141-reused",
		} {
			if strings.Contains(string(data), forbidden) {
				t.Fatalf("%s retains removed reference %q", path, forbidden)
			}
		}
	}
	if _, err := os.Stat("../../scripts/check-semstreams-state-cutover.sh"); !os.IsNotExist(err) {
		t.Fatalf("removed cutover script still exists or stat failed: %v", err)
	}
}

func TestDemoLauncherPreservesDurableNATSOnNormalStartup(t *testing.T) {
	tempDir := t.TempDir()
	semconnectRoot := filepath.Join(tempDir, "semconnect")
	if err := os.MkdirAll(filepath.Join(semconnectRoot, "conformance", ".vendor", "semstreams"), 0o755); err != nil {
		t.Fatalf("create fake SemConnect tree: %v", err)
	}
	if err := os.WriteFile(filepath.Join(semconnectRoot, "conformance", "compose.yml"), []byte("services: {}\n"), 0o644); err != nil {
		t.Fatalf("write fake SemConnect Compose file: %v", err)
	}

	binDir := filepath.Join(tempDir, "bin")
	if err := os.Mkdir(binDir, 0o755); err != nil {
		t.Fatalf("create fake bin directory: %v", err)
	}
	callLog := filepath.Join(tempDir, "docker-calls.log")
	fakeDocker := []byte("#!/usr/bin/env bash\nprintf '%s\\n' \"$*\" >> \"$DOCKER_CALL_LOG\"\n")
	if err := os.WriteFile(filepath.Join(binDir, "docker"), fakeDocker, 0o755); err != nil {
		t.Fatalf("write fake docker: %v", err)
	}

	cmd := exec.Command("bash", "../../scripts/demo-up.sh")
	cmd.Env = append(os.Environ(),
		"PATH="+binDir+string(os.PathListSeparator)+os.Getenv("PATH"),
		"DOCKER_CALL_LOG="+callLog,
		"SEMCONNECT_ROOT="+semconnectRoot,
		"COMPOSE_PROJECT_NAME=semlink-launcher-test",
		"SEMLINK_TAK_INBOUND_UDP_LISTEN=",
	)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("run demo launcher with fake Docker: %v\n%s", err, output)
	}

	data, err := os.ReadFile(callLog)
	if err != nil {
		t.Fatalf("read fake Docker call log: %v", err)
	}
	calls := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(calls) != 1 {
		t.Fatalf("docker call count = %d, want startup only; calls=%q", len(calls), calls)
	}
	if !strings.Contains(calls[0], "compose -p semlink-launcher-test") || !strings.Contains(calls[0], " up -d --build --wait nats semstreams-backend cs-api-server semlink-nats semlink") {
		t.Fatalf("Docker call is not the supported startup: %q", calls[0])
	}
	for _, destructive := range []string{" rm ", " down ", "--force-recreate semlink-nats", "-v"} {
		if strings.Contains(calls[0], destructive) {
			t.Fatalf("normal startup contains destructive NATS lifecycle operation %q: %q", destructive, calls[0])
		}
	}
}

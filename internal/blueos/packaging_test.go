package blueos

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBazaarMetadataSkeletonIsValid(t *testing.T) {
	path := filepath.Join("..", "..", "blueos", "extension", "metadata.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%s) error = %v", path, err)
	}
	var metadata map[string]string
	if err := json.Unmarshal(raw, &metadata); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	for _, key := range []string{"name", "website", "docker", "description"} {
		if metadata[key] == "" {
			t.Fatalf("metadata %q missing in %s", key, raw)
		}
	}
	if metadata["name"] != DefaultRegistration().Name {
		t.Fatalf("metadata name = %q, want %q", metadata["name"], DefaultRegistration().Name)
	}
}

func TestBlueOSComposeLoadsHandoffProfile(t *testing.T) {
	compose := readRepoText(t, "compose.blueos.yml")

	assertContainsAll(t, compose,
		"env_file:",
		"${SEMLINK_HANDOFF_PROFILE_FILE:-./configs/handoff/companion.env.example}",
		"SEMLINK_HANDOFF_PROFILE: \"/data/companion.env\"",
		"${SEMLINK_HANDOFF_PROFILE_FILE:-./configs/handoff/companion.env.example}:/data/companion.env:ro",
		"${SEMLINK_BLUEOS_HOST_PORT:-8081}:80",
	)
	if strings.Contains(compose, "SEMLINK_MAVLINK_UDP_LISTEN: \"${SEMLINK_MAVLINK_UDP_LISTEN:-}\"") {
		t.Fatal("compose.blueos.yml should not override the handoff profile MAVLink setting with an empty default")
	}
}

func TestBlueOSEntrypointLoadsMountedHandoffProfile(t *testing.T) {
	entrypoint := readRepoText(t, "docker", "blueos-extension", "entrypoint.sh")

	assertContainsAll(t, entrypoint,
		"SEMLINK_HANDOFF_PROFILE",
		"/data/companion.env",
		". \"$profile_path\"",
		"missing SemLink handoff profile",
	)
}

func TestBlueOSSmokeExportsHandoffProfileFile(t *testing.T) {
	script := readRepoText(t, "scripts", "blueos-extension-smoke.sh")

	assertContainsAll(t, script,
		"SEMLINK_HANDOFF_PROFILE_FILE",
		"configs/handoff/companion.env.example",
		". \"$SEMLINK_HANDOFF_PROFILE_FILE\"",
		"export SEMLINK_HANDOFF_PROFILE_FILE",
	)
}

func readRepoText(t *testing.T, parts ...string) string {
	t.Helper()

	path := filepath.Join(append([]string{"..", ".."}, parts...)...)
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%s) error = %v", path, err)
	}
	return string(raw)
}

func assertContainsAll(t *testing.T, text string, wants ...string) {
	t.Helper()

	for _, want := range wants {
		if !strings.Contains(text, want) {
			t.Fatalf("text does not contain %q", want)
		}
	}
}

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
	description := strings.ToLower(metadata["description"])
	for _, want := range []string{"mavlink", "companion", "mesh"} {
		if !strings.Contains(description, want) {
			t.Fatalf("metadata description %q does not declare %q", metadata["description"], want)
		}
	}
	assertTextLacks(t, description, "hardware transmit", "command transmit", "actuator")
}

func TestBlueOSDockerLabelsDeclareCompanionMeshWithoutHardwareTransmit(t *testing.T) {
	dockerfile := readRepoText(t, "docker", "blueos-extension", "Dockerfile")

	var company map[string]string
	if err := json.Unmarshal([]byte(extractDockerfileLabel(t, dockerfile, "company")), &company); err != nil {
		t.Fatalf("unmarshal company label: %v", err)
	}
	about := strings.ToLower(company["about"])
	for _, want := range []string{"mavlink", "companion", "mesh"} {
		if !strings.Contains(about, want) {
			t.Fatalf("company about %q does not declare %q", company["about"], want)
		}
	}

	var tags []string
	if err := json.Unmarshal([]byte(extractDockerfileLabel(t, dockerfile, "tags")), &tags); err != nil {
		t.Fatalf("unmarshal tags label: %v", err)
	}
	tagSet := make(map[string]bool, len(tags))
	for _, tag := range tags {
		tagSet[tag] = true
	}
	for _, want := range []string{"mavlink", "companion", "mesh", "navigation"} {
		if !tagSet[want] {
			t.Fatalf("tags = %#v, missing %q", tags, want)
		}
	}

	var permissions map[string]any
	if err := json.Unmarshal([]byte(extractDockerfileLabel(t, dockerfile, "permissions")), &permissions); err != nil {
		t.Fatalf("unmarshal permissions label: %v", err)
	}
	hostConfig, ok := permissions["HostConfig"].(map[string]any)
	if !ok {
		t.Fatalf("permissions HostConfig missing or wrong type: %#v", permissions)
	}
	if privileged, ok := hostConfig["Privileged"].(bool); ok && privileged {
		t.Fatalf("permissions declare privileged hardware access: %#v", hostConfig)
	}
	for _, forbiddenKey := range []string{"Devices", "CapAdd", "Privileged"} {
		if _, ok := hostConfig[forbiddenKey]; ok {
			t.Fatalf("permissions declare hardware command transmit capability via %s: %#v", forbiddenKey, hostConfig)
		}
	}
	assertTextLacks(t, strings.ToLower(dockerfile),
		"hardware-transmit-enabled",
		"command-transmit-enabled",
		"actuator-control",
		"/dev/serial",
		"/dev/tty",
	)
}

func TestBlueOSDockerfileDoesNotBundleDashboardUI(t *testing.T) {
	dockerfile := readRepoText(t, "docker", "blueos-extension", "Dockerfile")

	assertTextLacks(t, strings.ToLower(dockerfile),
		"from node",
		"npm ci",
		"npm run build",
		"ui/dist",
	)
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
		"-static=${SEMLINK_STATIC_DIR:-}",
	)
	if strings.Contains(entrypoint, "/app/ui/dist") {
		t.Fatal("BlueOS entrypoint should not serve bundled dashboard assets by default")
	}
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

func TestBlueOSSmokeVerifiesEvidenceContract(t *testing.T) {
	script := readRepoText(t, "scripts", "blueos-extension-smoke.sh")

	assertContainsAll(t, script,
		"/api/evidence",
		"c360.semlink.companion.evidence",
		"handoff node id",
		"SEMLINK_NODE_ID",
		"hardware_transmit_status",
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

func assertTextLacks(t *testing.T, text string, forbidden ...string) {
	t.Helper()

	for _, term := range forbidden {
		if strings.Contains(text, term) {
			t.Fatalf("text declares forbidden capability %q in %s", term, text)
		}
	}
}

func extractDockerfileLabel(t *testing.T, dockerfile, key string) string {
	t.Helper()

	prefix := key + "="
	index := strings.Index(dockerfile, prefix)
	if index < 0 {
		t.Fatalf("Dockerfile label %q missing", key)
	}
	valueStart := index + len(prefix)
	if valueStart >= len(dockerfile) {
		t.Fatalf("Dockerfile label %q has no value", key)
	}
	quote := dockerfile[valueStart]
	if quote != '\'' && quote != '"' {
		t.Fatalf("Dockerfile label %q is not quoted", key)
	}
	remainder := dockerfile[valueStart+1:]
	valueEnd := strings.IndexByte(remainder, quote)
	if valueEnd < 0 {
		t.Fatalf("Dockerfile label %q quote is not closed", key)
	}
	return remainder[:valueEnd]
}

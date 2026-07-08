package blueos

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestDefaultRegistrationIsBlueOSCompatible(t *testing.T) {
	registration := DefaultRegistration()
	if err := registration.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
	if registration.NewPage {
		t.Fatal("NewPage = true, want embedded BlueOS page")
	}
	if !registration.WorksInRelativePaths {
		t.Fatal("WorksInRelativePaths = false")
	}
	if registration.Extras["commands"] != "blocked-on-hardware" {
		t.Fatalf("commands extra = %q", registration.Extras["commands"])
	}
}

func TestDefaultRegistrationDeclaresCompanionMeshWithoutHardwareTransmit(t *testing.T) {
	registration := DefaultRegistration()
	description := strings.ToLower(registration.Description)
	for _, want := range []string{"mavlink", "companion", "mesh"} {
		if !strings.Contains(description, want) {
			t.Fatalf("description %q does not declare %q", registration.Description, want)
		}
	}
	if registration.Extras["runtime"] != "semlink-companion" {
		t.Fatalf("runtime extra = %q", registration.Extras["runtime"])
	}
	if registration.Extras["mavlink"] != "udp-ingress" {
		t.Fatalf("mavlink extra = %q", registration.Extras["mavlink"])
	}
	if registration.Extras["mesh"] != "summary-watermark-sync" {
		t.Fatalf("mesh extra = %q", registration.Extras["mesh"])
	}
	if registration.Extras["commands"] != "blocked-on-hardware" {
		t.Fatalf("commands extra = %q", registration.Extras["commands"])
	}
	forbidden := []string{"hardware-transmit-enabled", "command-transmit-enabled", "actuator-control"}
	serializedExtras, err := json.Marshal(registration.Extras)
	if err != nil {
		t.Fatalf("Marshal extras: %v", err)
	}
	lower := strings.ToLower(registration.Description + " " + string(serializedExtras))
	for _, term := range forbidden {
		if strings.Contains(lower, term) {
			t.Fatalf("registration declares forbidden capability %q in %s", term, lower)
		}
	}
}

func TestRegistrationJSONUsesBlueOSFieldNames(t *testing.T) {
	raw, err := json.Marshal(DefaultRegistration())
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	var body map[string]any
	if err := json.Unmarshal(raw, &body); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	for _, key := range []string{"name", "description", "icon", "company", "version", "webpage", "api", "new_page", "avoid_iframes", "works_in_relative_paths"} {
		if _, ok := body[key]; !ok {
			t.Fatalf("missing JSON key %q in %s", key, raw)
		}
	}
}

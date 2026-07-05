package blueos

import (
	"encoding/json"
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

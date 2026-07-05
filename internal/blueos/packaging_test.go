package blueos

import (
	"encoding/json"
	"os"
	"path/filepath"
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

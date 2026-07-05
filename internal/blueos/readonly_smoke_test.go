package blueos

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNavigatorSmokeScriptStaysReadOnly(t *testing.T) {
	path := filepath.Join("..", "..", "scripts", "navigator-readonly-smoke.sh")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%s) error = %v", path, err)
	}
	body := string(raw)
	for _, forbidden := range []string{
		"--request POST",
		"--request PUT",
		"--request PATCH",
		"--request DELETE",
		" -X POST",
		" -X PUT",
		" -X PATCH",
		" -X DELETE",
	} {
		if strings.Contains(body, forbidden) {
			t.Fatalf("read-only smoke script contains mutating request marker %q", forbidden)
		}
	}
	if !strings.Contains(body, "--request GET") {
		t.Fatal("read-only smoke script should make GET-only requests explicit")
	}
}

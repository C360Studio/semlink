package semstreams

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

const (
	expectedSemStreamsVersion = "v1.0.0-beta.160"
	expectedSemStreamsCommit  = "8403a2218000e45a31c5132fbfe01af42ed04f14"
	expectedSemStreamsSum     = "h1:sVkaLPOqs0Jy/iyYvfamkSi9An+Vq0o/5nqMxZVCvtk="
)

func TestSemStreamsDependencyPinAndReleaseEvidence(t *testing.T) {
	goMod, err := os.ReadFile("../../go.mod")
	if err != nil {
		t.Fatalf("read go.mod: %v", err)
	}
	moduleLine := "github.com/c360studio/semstreams " + expectedSemStreamsVersion
	if !strings.Contains(string(goMod), moduleLine) {
		t.Fatalf("go.mod does not contain exact pin %q", moduleLine)
	}
	goSum, err := os.ReadFile("../../go.sum")
	if err != nil {
		t.Fatalf("read go.sum: %v", err)
	}
	sumLine := moduleLine + " " + expectedSemStreamsSum
	if !strings.Contains(string(goSum), sumLine) {
		t.Fatalf("go.sum does not contain exact checksum %q", sumLine)
	}

	data, err := os.ReadFile("../../configs/dependencies/semstreams-beta160.json")
	if err != nil {
		t.Fatalf("read release evidence: %v", err)
	}
	var evidence struct {
		Module    string `json:"module"`
		Version   string `json:"version"`
		TagCommit string `json:"tag_commit"`
		ModuleSum string `json:"module_sum"`
	}
	if err := json.Unmarshal(data, &evidence); err != nil {
		t.Fatalf("decode release evidence: %v", err)
	}
	if evidence.Module != "github.com/c360studio/semstreams" || evidence.Version != expectedSemStreamsVersion ||
		evidence.TagCommit != expectedSemStreamsCommit || evidence.ModuleSum != expectedSemStreamsSum {
		t.Fatalf("SemStreams release evidence drifted: %#v", evidence)
	}
}

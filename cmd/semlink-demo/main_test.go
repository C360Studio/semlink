package main

import "testing"

func TestResolveSemLinkSourceRefPrefersExplicitValues(t *testing.T) {
	version, commit, err := resolveSemLinkSourceRef(
		"v0.3.0",
		"abc123",
		buildVCSInfo{version: "v0.2.0", revision: "def456"},
	)
	if err != nil {
		t.Fatalf("resolveSemLinkSourceRef() error = %v", err)
	}
	if version != "v0.3.0" || commit != "abc123" {
		t.Fatalf("source ref version=%q commit=%q", version, commit)
	}
}

func TestResolveSemLinkSourceRefUsesVCSRevision(t *testing.T) {
	version, commit, err := resolveSemLinkSourceRef(
		"",
		"",
		buildVCSInfo{version: "(devel)", revision: "0123456789abcdef"},
	)
	if err != nil {
		t.Fatalf("resolveSemLinkSourceRef() error = %v", err)
	}
	if version != "" || commit != "0123456789abcdef" {
		t.Fatalf("source ref version=%q commit=%q", version, commit)
	}
}

func TestResolveSemLinkSourceRefFallsBackToBuildVersion(t *testing.T) {
	version, commit, err := resolveSemLinkSourceRef(
		"",
		"",
		buildVCSInfo{version: "v1.2.3"},
	)
	if err != nil {
		t.Fatalf("resolveSemLinkSourceRef() error = %v", err)
	}
	if version != "v1.2.3" || commit != "" {
		t.Fatalf("source ref version=%q commit=%q", version, commit)
	}
}

func TestResolveSemLinkSourceRefRejectsMissingSourceRef(t *testing.T) {
	_, _, err := resolveSemLinkSourceRef("", "", buildVCSInfo{version: "(devel)"})
	if err == nil {
		t.Fatalf("resolveSemLinkSourceRef() error = nil, want missing source ref error")
	}
}

func TestDemoArtifactOptionsUsesResolvedVCSRevision(t *testing.T) {
	opts, err := demoArtifactOptions(
		"mesh",
		"ardurover",
		"",
		"",
		"",
		"",
		"",
		"",
		"",
		nil,
		buildVCSInfo{version: "(devel)", revision: "feedface"},
	)
	if err != nil {
		t.Fatalf("demoArtifactOptions() error = %v", err)
	}
	if opts.SemLinkVersion != "" || opts.SemLinkCommit != "feedface" {
		t.Fatalf("artifact source version=%q commit=%q", opts.SemLinkVersion, opts.SemLinkCommit)
	}
	if opts.GeneratorProfile != "mesh-deterministic" {
		t.Fatalf("generator profile = %q", opts.GeneratorProfile)
	}
}

func TestDemoArtifactOptionsPreservesExplicitGeneratorCommand(t *testing.T) {
	opts, err := demoArtifactOptions(
		"mesh",
		"ardurover",
		"",
		"",
		"",
		"",
		"semlink-demo -mode mesh -nodes 3 -vehicle-profile ardurover",
		"",
		"",
		nil,
		buildVCSInfo{version: "(devel)", revision: "feedface"},
	)
	if err != nil {
		t.Fatalf("demoArtifactOptions() error = %v", err)
	}
	if opts.GeneratorCommand != "semlink-demo -mode mesh -nodes 3 -vehicle-profile ardurover" {
		t.Fatalf("generator command = %q", opts.GeneratorCommand)
	}
}

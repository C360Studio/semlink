package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"runtime/debug"
	"strconv"
	"strings"
	"time"

	"github.com/c360studio/semlink/internal/e2e"
)

type artifactNodeSourceFlags []e2e.DemoArtifactNodeSource

func (f *artifactNodeSourceFlags) String() string {
	if f == nil {
		return ""
	}
	return fmt.Sprintf("%d node source(s)", len(*f))
}

func (f *artifactNodeSourceFlags) Set(value string) error {
	source, err := parseArtifactNodeSource(value)
	if err != nil {
		return err
	}
	*f = append(*f, source)
	return nil
}

func main() {
	mode := flag.String("mode", "single", "demo mode: single, mesh, sitl-artifact")
	output := flag.String("output", e2e.DefaultSingleNodeReportPath, "path for the generated JSON report")
	runtimeURL := flag.String(
		"runtime-url",
		"http://127.0.0.1:8081",
		"SemLink runtime base URL for sitl-artifact mode",
	)
	vehicleProfile := flag.String(
		"vehicle-profile",
		"ardurover",
		"vehicle profile label for the simulated MAVLink runtime",
	)
	nodes := flag.Int("nodes", 3, "number of local companion nodes for mesh mode")
	artifactOutput := flag.String(
		"artifact-output",
		"",
		"optional path for the generated companion demo artifact envelope",
	)
	artifactSourceFidelity := flag.String(
		"artifact-source-fidelity",
		e2e.ArtifactFidelityDeterministic,
		"artifact source fidelity: deterministic, sitl-backed, or hardware-adjacent",
	)
	artifactSemLinkVersion := flag.String(
		"artifact-semlink-version",
		"",
		"SemLink version recorded in artifact source metadata",
	)
	artifactSemLinkCommit := flag.String(
		"artifact-semlink-commit",
		"",
		"SemLink commit recorded in artifact source metadata",
	)
	artifactGeneratorProfile := flag.String(
		"artifact-generator-profile",
		"",
		"stable generator profile recorded in artifact source metadata",
	)
	artifactGeneratorCommand := flag.String(
		"artifact-generator-command",
		"",
		"sanitized generator command recorded in artifact source metadata",
	)
	artifactSimulatorFamily := flag.String(
		"artifact-simulator-family",
		"",
		"simulator family recorded in artifact source metadata",
	)
	artifactNoTransmitPosture := flag.String(
		"artifact-no-transmit-posture",
		"",
		"no-transmit posture recorded in artifact source metadata",
	)
	artifactVehicleSource := flag.String(
		"artifact-vehicle-source",
		e2e.DefaultSITLVehicleSource,
		"vehicle source recorded for SITL-backed artifact node metadata",
	)
	artifactRoute := flag.String(
		"artifact-route",
		"",
		"source route recorded for SITL-backed artifact node metadata",
	)
	var artifactNodeSources artifactNodeSourceFlags
	flag.Var(
		&artifactNodeSources,
		"artifact-node-source",
		"repeatable artifact node source metadata as key=value pairs",
	)
	flag.Parse()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	switch *mode {
	case "single":
		if *output == e2e.DefaultSingleNodeReportPath {
			*output = e2e.DefaultSingleNodeReportPath
		}
		report, err := e2e.RunSingleNodeDemo(ctx, e2e.SingleNodeDemoConfig{
			VehicleProfile: *vehicleProfile,
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "single-node demo failed: %v\n", err)
			os.Exit(1)
		}
		if err := e2e.WriteJSONReport(*output, report); err != nil {
			fmt.Fprintf(os.Stderr, "write report: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("single-node demo report: %s\n", *output)
		if *artifactOutput != "" {
			opts, err := demoArtifactOptions(
				"single",
				*vehicleProfile,
				*artifactSourceFidelity,
				*artifactSemLinkVersion,
				*artifactSemLinkCommit,
				*artifactGeneratorProfile,
				*artifactGeneratorCommand,
				*artifactSimulatorFamily,
				*artifactNoTransmitPosture,
				artifactNodeSources,
				currentBuildVCSInfo(ctx),
			)
			if err != nil {
				fmt.Fprintf(os.Stderr, "build single-node artifact metadata: %v\n", err)
				os.Exit(1)
			}
			artifact, err := e2e.BuildSingleNodeDemoArtifact(report, opts)
			if err != nil {
				fmt.Fprintf(os.Stderr, "build single-node artifact: %v\n", err)
				os.Exit(1)
			}
			if err := e2e.WriteJSONReport(*artifactOutput, artifact); err != nil {
				fmt.Fprintf(os.Stderr, "write artifact: %v\n", err)
				os.Exit(1)
			}
			fmt.Printf("single-node demo artifact: %s\n", *artifactOutput)
		}
	case "mesh":
		if *output == e2e.DefaultSingleNodeReportPath {
			*output = e2e.DefaultSimpleMeshReportPath
		}
		report, err := e2e.RunSimpleMeshDemo(ctx, e2e.SimpleMeshDemoConfig{
			Nodes:          *nodes,
			VehicleProfile: *vehicleProfile,
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "simple mesh demo failed: %v\n", err)
			os.Exit(1)
		}
		if err := e2e.WriteJSONReport(*output, report); err != nil {
			fmt.Fprintf(os.Stderr, "write report: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("simple mesh demo report: %s\n", *output)
		if *artifactOutput != "" {
			generatorCommand := *artifactGeneratorCommand
			if strings.TrimSpace(generatorCommand) == "" {
				generatorCommand = fmt.Sprintf(
					"semlink-demo -mode mesh -nodes %d -vehicle-profile %s",
					*nodes,
					*vehicleProfile,
				)
			}
			opts, err := demoArtifactOptions(
				"mesh",
				*vehicleProfile,
				*artifactSourceFidelity,
				*artifactSemLinkVersion,
				*artifactSemLinkCommit,
				*artifactGeneratorProfile,
				generatorCommand,
				*artifactSimulatorFamily,
				*artifactNoTransmitPosture,
				artifactNodeSources,
				currentBuildVCSInfo(ctx),
			)
			if err != nil {
				fmt.Fprintf(os.Stderr, "build simple mesh artifact metadata: %v\n", err)
				os.Exit(1)
			}
			artifact, err := e2e.BuildSimpleMeshDemoArtifact(report, opts)
			if err != nil {
				fmt.Fprintf(os.Stderr, "build simple mesh artifact: %v\n", err)
				os.Exit(1)
			}
			if err := e2e.WriteJSONReport(*artifactOutput, artifact); err != nil {
				fmt.Fprintf(os.Stderr, "write artifact: %v\n", err)
				os.Exit(1)
			}
			fmt.Printf("simple mesh demo artifact: %s\n", *artifactOutput)
		}
	case "sitl-artifact":
		if *output == e2e.DefaultSingleNodeReportPath {
			*output = e2e.DefaultSITLBackedReportPath
		}
		if *artifactOutput == "" {
			*artifactOutput = e2e.DefaultSITLBackedArtifactPath
		}
		if strings.TrimSpace(*artifactSourceFidelity) != "" &&
			strings.TrimSpace(*artifactSourceFidelity) != e2e.ArtifactFidelityDeterministic &&
			strings.TrimSpace(*artifactSourceFidelity) != e2e.ArtifactFidelitySITLBacked {
			fmt.Fprintf(os.Stderr, "sitl-artifact mode only emits %q fidelity\n", e2e.ArtifactFidelitySITLBacked)
			os.Exit(2)
		}
		client := &http.Client{Timeout: 2 * time.Second}
		evidenceProbe, evidence, err := e2e.FetchRuntimeEvidence(ctx, client, *runtimeURL)
		if err != nil {
			fmt.Fprintf(os.Stderr, "fetch runtime evidence: %v\n", err)
			os.Exit(1)
		}
		resolvedVersion, resolvedCommit, err := resolveSemLinkSourceRef(
			*artifactSemLinkVersion,
			*artifactSemLinkCommit,
			currentBuildVCSInfo(ctx),
		)
		if err != nil {
			fmt.Fprintf(os.Stderr, "resolve artifact source ref: %v\n", err)
			os.Exit(1)
		}
		generatorCommand := strings.TrimSpace(*artifactGeneratorCommand)
		if generatorCommand == "" {
			generatorCommand = fmt.Sprintf(
				"semlink-demo -mode sitl-artifact -runtime-url %s -vehicle-profile %s",
				*runtimeURL,
				*vehicleProfile,
			)
		}
		report, artifact, err := e2e.BuildSITLBackedSingleNodeDemoArtifact(e2e.SITLBackedDemoConfig{
			Evidence:          evidence,
			EvidenceProbe:     evidenceProbe,
			RuntimeBaseURL:    *runtimeURL,
			VehicleProfile:    *vehicleProfile,
			SimulatorFamily:   *artifactSimulatorFamily,
			VehicleSource:     *artifactVehicleSource,
			Route:             *artifactRoute,
			SemLinkVersion:    resolvedVersion,
			SemLinkCommit:     resolvedCommit,
			GeneratorCommand:  generatorCommand,
			GeneratorProfile:  *artifactGeneratorProfile,
			NoTransmitPosture: *artifactNoTransmitPosture,
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "build SITL-backed artifact: %v\n", err)
			os.Exit(1)
		}
		if err := e2e.WriteJSONReport(*output, report); err != nil {
			fmt.Fprintf(os.Stderr, "write report: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("SITL-backed demo report: %s\n", *output)
		if err := e2e.WriteJSONReport(*artifactOutput, artifact); err != nil {
			fmt.Fprintf(os.Stderr, "write artifact: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("SITL-backed demo artifact: %s\n", *artifactOutput)
	default:
		fmt.Fprintf(os.Stderr, "unsupported demo mode %q\n", *mode)
		os.Exit(2)
	}
}

func demoArtifactOptions(
	mode string,
	vehicleProfile string,
	sourceFidelity string,
	semlinkVersion string,
	semlinkCommit string,
	generatorProfile string,
	generatorCommand string,
	simulatorFamily string,
	noTransmitPosture string,
	nodeSources artifactNodeSourceFlags,
	buildInfo buildVCSInfo,
) (e2e.DemoArtifactOptions, error) {
	if strings.TrimSpace(generatorProfile) == "" {
		generatorProfile = mode + "-deterministic"
	}
	if strings.TrimSpace(generatorCommand) == "" {
		generatorCommand = fmt.Sprintf("semlink-demo -mode %s -vehicle-profile %s", mode, vehicleProfile)
	}
	resolvedVersion, resolvedCommit, err := resolveSemLinkSourceRef(semlinkVersion, semlinkCommit, buildInfo)
	if err != nil {
		return e2e.DemoArtifactOptions{}, err
	}
	return e2e.DemoArtifactOptions{
		SourceFidelity:    sourceFidelity,
		SemLinkVersion:    resolvedVersion,
		SemLinkCommit:     resolvedCommit,
		GeneratorCommand:  generatorCommand,
		GeneratorProfile:  generatorProfile,
		SimulatorFamily:   simulatorFamily,
		NoTransmitPosture: noTransmitPosture,
		Nodes:             append([]e2e.DemoArtifactNodeSource(nil), nodeSources...),
	}, nil
}

func parseArtifactNodeSource(value string) (e2e.DemoArtifactNodeSource, error) {
	fields := strings.Split(value, ",")
	source := e2e.DemoArtifactNodeSource{}
	for _, field := range fields {
		key, raw, ok := strings.Cut(field, "=")
		if !ok {
			return e2e.DemoArtifactNodeSource{}, fmt.Errorf("artifact node source field %q must be key=value", field)
		}
		key = strings.TrimSpace(key)
		raw = strings.TrimSpace(raw)
		switch key {
		case "node_id":
			source.NodeID = raw
		case "source_fidelity":
			source.SourceFidelity = raw
		case "simulator_family":
			source.SimulatorFamily = raw
		case "vehicle_source":
			source.VehicleSource = raw
		case "mavlink_system_id":
			systemID, err := strconv.Atoi(raw)
			if err != nil {
				return e2e.DemoArtifactNodeSource{}, fmt.Errorf("parse mavlink_system_id %q: %w", raw, err)
			}
			source.MAVLinkSystemID = systemID
		case "route":
			source.Route = raw
		default:
			return e2e.DemoArtifactNodeSource{}, fmt.Errorf("unsupported artifact node source key %q", key)
		}
	}
	return source, nil
}

type buildVCSInfo struct {
	version  string
	revision string
}

func currentBuildVCSInfo(ctx context.Context) buildVCSInfo {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return buildVCSInfo{revision: gitHeadRevision(ctx)}
	}
	out := buildVCSInfo{version: strings.TrimSpace(info.Main.Version)}
	for _, setting := range info.Settings {
		if setting.Key == "vcs.revision" {
			out.revision = strings.TrimSpace(setting.Value)
			break
		}
	}
	if out.revision == "" {
		out.revision = gitHeadRevision(ctx)
	}
	return out
}

func gitHeadRevision(ctx context.Context) string {
	if ctx == nil {
		ctx = context.Background()
	}
	output, err := exec.CommandContext(ctx, "git", "rev-parse", "--verify", "HEAD").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(output))
}

func resolveSemLinkSourceRef(explicitVersion, explicitCommit string, info buildVCSInfo) (string, string, error) {
	explicitVersion = strings.TrimSpace(explicitVersion)
	explicitCommit = strings.TrimSpace(explicitCommit)
	if explicitVersion != "" || explicitCommit != "" {
		return explicitVersion, explicitCommit, nil
	}
	if info.revision != "" {
		return "", info.revision, nil
	}
	if version := strings.TrimSpace(info.version); version != "" && version != "(devel)" {
		return version, "", nil
	}
	return "", "", fmt.Errorf(
		"artifact output requires a real semlink source ref; " +
			"set -artifact-semlink-commit or -artifact-semlink-version",
	)
}

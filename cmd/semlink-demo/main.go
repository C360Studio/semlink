package main

import (
	"context"
	"flag"
	"fmt"
	"os"
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
	mode := flag.String("mode", "single", "demo mode: single")
	output := flag.String("output", e2e.DefaultSingleNodeReportPath, "path for the generated JSON report")
	vehicleProfile := flag.String(
		"vehicle-profile",
		"ardurover",
		"vehicle profile label for the simulated MAVLink runtime",
	)
	nodes := flag.Int("nodes", 3, "number of local companion nodes for mesh mode")
	vehiclesPerNode := flag.Int("vehicles-per-node", 1, "number of simulated MAVLink vehicles per companion node")
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
		e2e.DefaultSemLinkArtifactVersion,
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
			opts := demoArtifactOptions(
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
			)
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
			Nodes:           *nodes,
			VehiclesPerNode: *vehiclesPerNode,
			VehicleProfile:  *vehicleProfile,
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
			opts := demoArtifactOptions(
				"mesh",
				*vehicleProfile,
				*artifactSourceFidelity,
				*artifactSemLinkVersion,
				*artifactSemLinkCommit,
				*artifactGeneratorProfile,
				*artifactGeneratorCommand,
				*artifactSimulatorFamily,
				*artifactNoTransmitPosture,
				artifactNodeSources,
			)
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
) e2e.DemoArtifactOptions {
	if strings.TrimSpace(generatorProfile) == "" {
		generatorProfile = mode + "-deterministic"
	}
	if strings.TrimSpace(generatorCommand) == "" {
		generatorCommand = fmt.Sprintf("semlink-demo -mode %s -vehicle-profile %s", mode, vehicleProfile)
	}
	return e2e.DemoArtifactOptions{
		SourceFidelity:    sourceFidelity,
		SemLinkVersion:    semlinkVersion,
		SemLinkCommit:     semlinkCommit,
		GeneratorCommand:  generatorCommand,
		GeneratorProfile:  generatorProfile,
		SimulatorFamily:   simulatorFamily,
		NoTransmitPosture: noTransmitPosture,
		Nodes:             append([]e2e.DemoArtifactNodeSource(nil), nodeSources...),
	}
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

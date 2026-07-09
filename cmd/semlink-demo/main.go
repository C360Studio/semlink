package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/c360studio/semlink/internal/e2e"
)

func main() {
	mode := flag.String("mode", "single", "demo mode: single")
	output := flag.String("output", e2e.DefaultSingleNodeReportPath, "path for the generated JSON report")
	vehicleProfile := flag.String("vehicle-profile", "ardurover", "vehicle profile label for the simulated MAVLink runtime")
	nodes := flag.Int("nodes", 3, "number of local companion nodes for mesh mode")
	vehiclesPerNode := flag.Int("vehicles-per-node", 1, "number of simulated MAVLink vehicles per companion node")
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
	default:
		fmt.Fprintf(os.Stderr, "unsupported demo mode %q\n", *mode)
		os.Exit(2)
	}
}

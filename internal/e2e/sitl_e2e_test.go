//go:build !windows

package e2e

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/c360studio/semlink/internal/gcs"
	"github.com/c360studio/semlink/internal/handoff"
	semruntime "github.com/c360studio/semlink/internal/semstreams"
	"github.com/c360studio/semlink/internal/sitl"
)

func TestArduPilotSITLArtifactE2E(t *testing.T) {
	if !envEnabled("SEMLINK_E2E_SITL") {
		t.Skip("set SEMLINK_E2E_SITL=1 to run the real ArduPilot SITL artifact e2e")
	}
	launcher := sitlLauncherFromEnv(t)

	timeout := envDuration("SEMLINK_E2E_SITL_TIMEOUT", 2*time.Minute)
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	udpPort := envInt("SEMLINK_E2E_SITL_UDP_PORT", freeUDPPort(t))
	udpListenHost := envString("SEMLINK_E2E_SITL_UDP_LISTEN_HOST", launcher.defaultListenHost())
	outputHost := envString("SEMLINK_E2E_SITL_OUTPUT_HOST", launcher.defaultOutputHost())
	profile := sitlE2EProfile(t, udpListenHost, outputHost, udpPort)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	rt, err := semruntime.StartRuntime(ctx, semruntime.RuntimeOptions{
		Embedded: true,
		Logger:   logger,
	})
	if err != nil {
		t.Fatalf("StartRuntime() error = %v", err)
	}
	t.Cleanup(func() {
		stopCtx, stopCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer stopCancel()
		if err := rt.Stop(stopCtx); err != nil {
			t.Logf("runtime stop failed: %v", err)
		}
	})

	store := gcs.NewStore(rt.NATSURL, true)
	demo, err := gcs.NewDemo(gcs.DemoConfig{
		Vehicles:         1,
		Hz:               5,
		BufferCapacity:   10_000,
		MAVLinkUDPListen: profile.MAVLinkUDPListen,
		Logger:           logger,
	}, rt, store)
	if err != nil {
		t.Fatalf("NewDemo() error = %v", err)
	}
	demo.Start(ctx)

	server := httptest.NewServer(gcs.NewServer(
		store,
		gcs.NewCommandService(rt.Graph, store),
		"",
		gcs.ServerOptions{
			Graph:          rt.Graph,
			NodeID:         profile.NodeID,
			HandoffProfile: &profile,
		},
	).Handler())
	t.Cleanup(server.Close)

	lane := sitl.DefaultArduRoverLane()
	lane.OutputHost = outputHost
	lane.OutputPort = udpPort
	lane.Frame = sitl.Frame(envString("SEMLINK_E2E_SITL_FRAME", string(sitl.FrameRover)))
	lane.Aircraft = envString("SEMLINK_E2E_SITL_AIRCRAFT", "semlink-e2e-ardurover")
	lane.Speedup = envInt("SEMLINK_E2E_SITL_SPEEDUP", 1)
	lane.WipeEEPROM = envEnabled("SEMLINK_E2E_SITL_WIPE")
	if launcher.mode == "docker" && !envEnabled("SEMLINK_E2E_SITL_DIRECT_SERIAL") {
		lane.NoMAVProxy = false
		lane.NoExtraPorts = true
		lane.MAVProxyOut = fmt.Sprintf("%s:%d", outputHost, udpPort)
		lane.MAVProxyArgs = envString(
			"SEMLINK_E2E_SITL_MAVPROXY_ARGS",
			"--non-interactive --default-modules=output --retries=30",
		)
		lane.DelayStartSeconds = envInt("SEMLINK_E2E_SITL_DELAY_START_SECONDS", 5)
	}
	args, err := lane.SimVehicleArgs()
	if err != nil {
		t.Fatalf("SITL lane args: %v", err)
	}

	cmd, varName := launcher.command(args)
	t.Logf("starting ArduPilot SITL via %s: %s %s", varName, cmd.Path, strings.Join(cmd.Args[1:], " "))
	var simOutput bytes.Buffer
	cmd.Stdout = &simOutput
	cmd.Stderr = &simOutput
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if err := cmd.Start(); err != nil {
		t.Fatalf("start sim_vehicle.py: %v", err)
	}
	simDone := make(chan error, 1)
	go func() {
		simDone <- cmd.Wait()
		close(simDone)
	}()
	t.Cleanup(func() {
		stopDockerContainer(t, launcher.cidFile)
		terminateProcessGroup(t, cmd, &simOutput, simDone)
	})

	client := &http.Client{Timeout: 2 * time.Second}
	semlinkCommit := resolveE2ESemLinkCommit(t, ctx)
	report, artifact := waitForSITLArtifact(t, ctx, client, server.URL, semlinkCommit, simDone, &simOutput)

	outDir := t.TempDir()
	reportPath := filepath.Join(outDir, "report.json")
	artifactPath := filepath.Join(outDir, "artifact.json")
	if err := WriteJSONReport(reportPath, report); err != nil {
		t.Fatalf("write SITL report: %v", err)
	}
	if err := WriteJSONReport(artifactPath, artifact); err != nil {
		t.Fatalf("write SITL artifact: %v", err)
	}

	if artifact.Source.SourceFidelity != ArtifactFidelitySITLBacked {
		t.Fatalf("source fidelity = %q", artifact.Source.SourceFidelity)
	}
	if artifact.Source.SemLinkCommit != semlinkCommit {
		t.Fatalf("semlink commit = %q, want %q", artifact.Source.SemLinkCommit, semlinkCommit)
	}
	if len(artifact.Source.Nodes) != 1 || artifact.Source.Nodes[0].MAVLinkSystemID <= 0 {
		t.Fatalf("artifact node metadata = %#v", artifact.Source.Nodes)
	}
	t.Logf("SITL-backed artifact written: %s", artifactPath)
}

type sitlLauncher struct {
	mode     string
	binary   string
	image    string
	platform string
	network  string
	cidFile  string
}

func sitlLauncherFromEnv(t *testing.T) sitlLauncher {
	t.Helper()
	if image := strings.TrimSpace(os.Getenv("SEMLINK_E2E_SITL_DOCKER_IMAGE")); image != "" {
		if _, err := exec.LookPath("docker"); err != nil {
			t.Fatalf("SEMLINK_E2E_SITL_DOCKER_IMAGE requires docker on PATH: %v", err)
		}
		return sitlLauncher{
			mode:     "docker",
			image:    image,
			platform: strings.TrimSpace(os.Getenv("SEMLINK_E2E_SITL_DOCKER_PLATFORM")),
			network:  strings.TrimSpace(os.Getenv("SEMLINK_E2E_SITL_DOCKER_NETWORK")),
			cidFile:  filepath.Join(t.TempDir(), "sitl.cid"),
		}
	}
	simVehicle := strings.TrimSpace(os.Getenv("SIM_VEHICLE"))
	if simVehicle == "" {
		var err error
		simVehicle, err = exec.LookPath("sim_vehicle.py")
		if err != nil {
			t.Fatalf(
				"SEMLINK_E2E_SITL=1 requires sim_vehicle.py on PATH, SIM_VEHICLE, or SEMLINK_E2E_SITL_DOCKER_IMAGE: %v",
				err,
			)
		}
	}
	return sitlLauncher{mode: "host", binary: simVehicle}
}

func (l sitlLauncher) command(simVehicleArgs []string) (*exec.Cmd, string) {
	switch l.mode {
	case "docker":
		args := []string{"run", "--rm"}
		if l.cidFile != "" {
			args = append(args, "--cidfile", l.cidFile)
		}
		if l.platform != "" {
			args = append(args, "--platform", l.platform)
		}
		if l.network != "" {
			args = append(args, "--network", l.network)
		}
		args = append(args, l.image)
		args = append(args, simVehicleArgs...)
		return exec.Command("docker", args...), "SEMLINK_E2E_SITL_DOCKER_IMAGE"
	default:
		return exec.Command(l.binary, simVehicleArgs...), "SIM_VEHICLE"
	}
}

func (l sitlLauncher) defaultListenHost() string {
	if l.mode == "docker" {
		return "0.0.0.0"
	}
	return "127.0.0.1"
}

func (l sitlLauncher) defaultOutputHost() string {
	if l.mode == "docker" {
		return "host.docker.internal"
	}
	return "127.0.0.1"
}

func waitForSITLArtifact(
	t *testing.T,
	ctx context.Context,
	client *http.Client,
	runtimeURL string,
	semlinkCommit string,
	simDone <-chan error,
	simOutput *bytes.Buffer,
) (SingleNodeDemoReport, DemoArtifact) {
	t.Helper()
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	var lastErr error
	var lastLog time.Time
	for {
		probe, evidence, err := FetchRuntimeEvidence(ctx, client, runtimeURL)
		if err == nil {
			report, artifact, buildErr := BuildSITLBackedSingleNodeDemoArtifact(SITLBackedDemoConfig{
				Evidence:         evidence,
				EvidenceProbe:    probe,
				RuntimeBaseURL:   runtimeURL,
				VehicleProfile:   "ardurover",
				VehicleSource:    "ArduPilot Rover SITL without Gazebo",
				SemLinkCommit:    semlinkCommit,
				GeneratorCommand: "go test ./internal/e2e -run TestArduPilotSITLArtifactE2E",
				GeneratorProfile: "sitl-e2e",
			})
			if buildErr == nil {
				return report, artifact
			}
			lastErr = buildErr
		} else {
			lastErr = err
		}

		select {
		case err, ok := <-simDone:
			if ok {
				t.Fatalf("sim_vehicle.py exited before SITL-backed evidence: %v\n%s", err, tailString(simOutput.String(), 12_000))
			}
			t.Fatalf("sim_vehicle.py exited before SITL-backed evidence\n%s", tailString(simOutput.String(), 12_000))
		case <-ctx.Done():
			t.Fatalf("timed out waiting for SITL-backed evidence: %v", lastErr)
		case <-ticker.C:
			if time.Since(lastLog) >= 10*time.Second {
				t.Logf("waiting for SITL-backed evidence: %v", lastErr)
				lastLog = time.Now()
			}
		}
	}
}

func sitlE2EProfile(t *testing.T, udpListenHost string, udpOutputHost string, udpPort int) handoff.Profile {
	t.Helper()
	profile, err := handoff.ParseEnv(map[string]string{
		handoff.EnvNodeID:                  "semlink-sitl-e2e",
		handoff.EnvVehicleID:               "vehicle:sitl",
		handoff.EnvCallsign:                "SITL-E2E",
		handoff.EnvHTTPListen:              "127.0.0.1:8081",
		handoff.EnvEmbeddedNATS:            "true",
		handoff.EnvNATSURL:                 "nats://embedded-sitl-e2e",
		handoff.EnvMAVLinkUDPListen:        fmt.Sprintf("%s:%d", udpListenHost, udpPort),
		handoff.EnvMAVLinkUDPHost:          udpOutputHost,
		handoff.EnvMAVLinkUDPPort:          strconv.Itoa(udpPort),
		handoff.EnvVehicles:                "1",
		handoff.EnvHz:                      "5",
		handoff.EnvBuffer:                  "10000",
		handoff.EnvCommandRuntimeMode:      string(handoff.CommandRuntimeHardwareReadonly),
		handoff.EnvHardwareTransmitEnabled: "false",
	})
	if err != nil {
		t.Fatalf("ParseEnv() error = %v", err)
	}
	return profile
}

func freeUDPPort(t *testing.T) int {
	t.Helper()
	addr := &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 0}
	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		t.Fatalf("allocate UDP port: %v", err)
	}
	defer conn.Close()
	return conn.LocalAddr().(*net.UDPAddr).Port
}

func resolveE2ESemLinkCommit(t *testing.T, ctx context.Context) string {
	t.Helper()
	if value := strings.TrimSpace(os.Getenv("SEMLINK_E2E_SITL_SEMLINK_COMMIT")); value != "" {
		return value
	}
	output, err := exec.CommandContext(ctx, "git", "rev-parse", "--verify", "HEAD").Output()
	if err != nil {
		t.Fatalf("resolve SemLink commit for SITL e2e artifact: %v", err)
	}
	return strings.TrimSpace(string(output))
}

func terminateProcessGroup(t *testing.T, cmd *exec.Cmd, output *bytes.Buffer, done <-chan error) {
	t.Helper()
	if cmd.Process == nil {
		return
	}
	select {
	case err, ok := <-done:
		if ok && err != nil && !isExpectedCleanupExit(err) {
			t.Logf("sim_vehicle.py exited: %v\n%s", err, tailString(output.String(), 12_000))
		}
		return
	default:
	}
	_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGTERM)
	select {
	case err, ok := <-done:
		if ok && err != nil && !isExpectedCleanupExit(err) {
			t.Logf("sim_vehicle.py exited: %v\n%s", err, tailString(output.String(), 12_000))
		}
	case <-time.After(5 * time.Second):
		_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
		err, ok := <-done
		if ok && err != nil && !isExpectedCleanupExit(err) {
			t.Logf("sim_vehicle.py killed after cleanup timeout: %v\n%s", err, tailString(output.String(), 12_000))
		}
	}
}

func stopDockerContainer(t *testing.T, cidFile string) {
	t.Helper()
	if cidFile == "" {
		return
	}
	body, err := os.ReadFile(cidFile)
	if err != nil {
		if !os.IsNotExist(err) {
			t.Logf("read SITL docker cidfile: %v", err)
		}
		return
	}
	containerID := strings.TrimSpace(string(body))
	if containerID == "" {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	output, err := exec.CommandContext(ctx, "docker", "stop", containerID).CombinedOutput()
	if err != nil && !strings.Contains(string(output), "No such container") {
		t.Logf("stop SITL docker container %s: %v\n%s", containerID, err, tailString(string(output), 4_000))
	}
}

func isExpectedCleanupExit(err error) bool {
	if err == nil {
		return true
	}
	message := err.Error()
	return strings.Contains(message, "signal: terminated") ||
		strings.Contains(message, "signal: killed") ||
		strings.Contains(message, "exit status 137") ||
		strings.Contains(message, "exit status 143")
}

func tailString(value string, maxBytes int) string {
	if maxBytes <= 0 || len(value) <= maxBytes {
		return value
	}
	return value[len(value)-maxBytes:]
}

func envString(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

func envInt(key string, fallback int) int {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		return fallback
	}
	return value
}

func envDuration(key string, fallback time.Duration) time.Duration {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback
	}
	value, err := time.ParseDuration(raw)
	if err != nil {
		return fallback
	}
	return value
}

func envEnabled(key string) bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv(key))) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

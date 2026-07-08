package gcs

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/c360studio/semlink/internal/handoff"
	"github.com/c360studio/semlink/internal/mavlink"
	"github.com/c360studio/semlink/internal/projector"
)

func TestDemoConfigSelectsExternalMAVLinkInsteadOfSimulator(t *testing.T) {
	if got := (DemoConfig{}).TelemetrySourceMode(); got != TelemetrySourceInternalSimulator {
		t.Fatalf("empty config source mode = %q", got)
	}
	if got := (DemoConfig{MAVLinkUDPListen: "127.0.0.1:14550"}).TelemetrySourceMode(); got != TelemetrySourceExternalMAVLinkUDP {
		t.Fatalf("UDP config source mode = %q", got)
	}
}

func TestUDPEvidenceSmokeProjectsExternalMAVLinkState(t *testing.T) {
	observedAt := time.Unix(200, 0).UTC()
	source, err := mavlink.ListenUDP(mavlink.UDPSourceConfig{
		ListenAddr:    "127.0.0.1:0",
		SubjectPrefix: "mavlink.raw.ardupilot-sitl",
		Clock:         func() time.Time { return observedAt },
	})
	if err != nil {
		t.Fatalf("ListenUDP() error = %v", err)
	}
	defer source.Close()

	conn, err := net.Dial("udp", source.LocalAddr().String())
	if err != nil {
		t.Fatalf("Dial() error = %v", err)
	}
	defer conn.Close()

	frame, err := mavlink.EncodeV2(12, 42, 1, mavlink.MessageHeartbeat,
		mavlink.HeartbeatPayloadForType(true, 0, mavlink.MavTypeSurfaceBoat))
	if err != nil {
		t.Fatalf("EncodeV2() error = %v", err)
	}
	if _, err := conn.Write(frame); err != nil {
		t.Fatalf("Write() error = %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	raw, err := source.Read(ctx)
	if err != nil {
		t.Fatalf("Read() error = %v", err)
	}

	msg, err := mavlink.DecodeMessage(raw.Bytes)
	if err != nil {
		t.Fatalf("DecodeMessage() error = %v", err)
	}
	proj := projector.New(projector.DefaultConfig())
	if projections := proj.Apply(msg, raw.EmittedAt); len(projections) == 0 {
		t.Fatal("external MAVLink heartbeat produced no projection")
	}
	vehicles, alerts := proj.Snapshot()

	store := NewStore("nats://demo", true)
	store.RecordRawFrame()
	store.RecordDecodedFrame()
	store.RecordBuffer(0, 10_000, 0)
	store.ApplyProjectorSnapshot(vehicles, alerts)

	addr, ok := source.LocalAddr().(*net.UDPAddr)
	if !ok {
		t.Fatalf("local addr type = %T", source.LocalAddr())
	}
	profile, err := handoff.ParseEnv(map[string]string{
		handoff.EnvNodeID:                  "boat-alpha",
		handoff.EnvVehicleID:               projector.VehicleEntityID(42),
		handoff.EnvCallsign:                "BOAT-042",
		handoff.EnvHTTPListen:              ":8081",
		handoff.EnvEmbeddedNATS:            "true",
		handoff.EnvNATSURL:                 "nats://demo",
		handoff.EnvMAVLinkUDPListen:        addr.String(),
		handoff.EnvMAVLinkUDPHost:          addr.IP.String(),
		handoff.EnvMAVLinkUDPPort:          strconv.Itoa(addr.Port),
		handoff.EnvVehicles:                "1",
		handoff.EnvHz:                      "5",
		handoff.EnvBuffer:                  "10000",
		handoff.EnvCommandRuntimeMode:      string(handoff.CommandRuntimeHardwareReadonly),
		handoff.EnvHardwareTransmitEnabled: "false",
	})
	if err != nil {
		t.Fatalf("ParseEnv() error = %v", err)
	}

	server := NewServer(store, nil, "", ServerOptions{
		NodeID:         profile.NodeID,
		HandoffProfile: &profile,
	})
	req := httptest.NewRequest(http.MethodGet, "/api/evidence", nil)
	rr := httptest.NewRecorder()
	server.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", rr.Code, rr.Body.String())
	}

	var body EvidenceBundle
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode evidence bundle: %v", err)
	}
	if !body.Profile.MAVLink.ExternalInputConfigured {
		t.Fatalf("profile MAVLink = %#v", body.Profile.MAVLink)
	}
	if body.Profile.Simulator.Enabled || body.Profile.Simulator.Source != string(TelemetrySourceExternalMAVLinkUDP) {
		t.Fatalf("profile simulator = %#v", body.Profile.Simulator)
	}
	if body.Node.RawFrames != 1 || body.Node.DecodedFrames != 1 {
		t.Fatalf("node frame metrics = raw:%d decoded:%d", body.Node.RawFrames, body.Node.DecodedFrames)
	}
	if len(body.Vehicles) != 1 {
		t.Fatalf("vehicles = %#v", body.Vehicles)
	}
	vehicle := body.Vehicles[0]
	if vehicle.SystemID != 42 || vehicle.VehicleType != "surface-boat" || vehicle.LinkStatus != "online" {
		t.Fatalf("vehicle evidence = %#v", vehicle)
	}
	if vehicle.EvidenceClass != "mavlink-current-state" {
		t.Fatalf("evidence class = %q", vehicle.EvidenceClass)
	}
}

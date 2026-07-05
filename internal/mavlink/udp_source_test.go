package mavlink

import (
	"context"
	"net"
	"testing"
	"time"
)

func TestUDPSourceReadsSupportedSITLFrame(t *testing.T) {
	source, err := ListenUDP(UDPSourceConfig{
		ListenAddr:    "127.0.0.1:0",
		SubjectPrefix: "mavlink.raw.ardupilot-sitl",
		Clock:         func() time.Time { return time.Unix(100, 0).UTC() },
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

	frame, err := EncodeV2(7, 42, 1, MessageHeartbeat, HeartbeatPayloadForType(true, 0, MavTypeGroundRover))
	if err != nil {
		t.Fatalf("EncodeV2() error = %v", err)
	}
	if _, err := conn.Write(frame); err != nil {
		t.Fatalf("Write() error = %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	got, err := source.Read(ctx)
	if err != nil {
		t.Fatalf("Read() error = %v", err)
	}

	if got.Subject != "mavlink.raw.ardupilot-sitl.sys-042" {
		t.Fatalf("Subject = %q", got.Subject)
	}
	if got.VehicleID != "sys-042" {
		t.Fatalf("VehicleID = %q", got.VehicleID)
	}
	if got.SystemID != 42 {
		t.Fatalf("SystemID = %d", got.SystemID)
	}
	if !got.EmittedAt.Equal(time.Unix(100, 0).UTC()) {
		t.Fatalf("EmittedAt = %s", got.EmittedAt)
	}

	msg, err := DecodeMessage(got.Bytes)
	if err != nil {
		t.Fatalf("DecodeMessage() error = %v", err)
	}
	heartbeat, ok := msg.(Heartbeat)
	if !ok {
		t.Fatalf("message type = %T, want Heartbeat", msg)
	}
	if heartbeat.Type != MavTypeGroundRover {
		t.Fatalf("heartbeat type = %d, want ground rover", heartbeat.Type)
	}
}

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

func TestUDPSourceBindsIPv4ForIPv4Wildcard(t *testing.T) {
	source, err := ListenUDP(UDPSourceConfig{ListenAddr: "0.0.0.0:0"})
	if err != nil {
		t.Fatalf("ListenUDP() error = %v", err)
	}
	defer source.Close()

	addr, ok := source.LocalAddr().(*net.UDPAddr)
	if !ok {
		t.Fatalf("LocalAddr() type = %T, want *net.UDPAddr", source.LocalAddr())
	}
	if addr.IP.To4() == nil {
		t.Fatalf("LocalAddr() = %v, want IPv4 socket", addr)
	}
}

func TestUDPSourceSplitsSupportedFramesFromOneDatagram(t *testing.T) {
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

	heartbeat, err := EncodeV2(7, 42, 1, MessageHeartbeat, HeartbeatPayloadForType(true, 0, MavTypeGroundRover))
	if err != nil {
		t.Fatalf("EncodeV2(heartbeat) error = %v", err)
	}
	sysStatus, err := EncodeV2(8, 42, 1, MessageSysStatus, SysStatusPayload(12000, 100, 95, 0))
	if err != nil {
		t.Fatalf("EncodeV2(sys status) error = %v", err)
	}
	packet := append(append([]byte{}, heartbeat...), sysStatus...)
	if _, err := conn.Write(packet); err != nil {
		t.Fatalf("Write() error = %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	first, err := source.Read(ctx)
	if err != nil {
		t.Fatalf("Read(first) error = %v", err)
	}
	second, err := source.Read(ctx)
	if err != nil {
		t.Fatalf("Read(second) error = %v", err)
	}

	if first.SystemID != 42 || second.SystemID != 42 {
		t.Fatalf("system ids = %d, %d; want both 42", first.SystemID, second.SystemID)
	}
	if _, err := DecodeMessage(first.Bytes); err != nil {
		t.Fatalf("DecodeMessage(first) error = %v", err)
	}
	msg, err := DecodeMessage(second.Bytes)
	if err != nil {
		t.Fatalf("DecodeMessage(second) error = %v", err)
	}
	if _, ok := msg.(SysStatus); !ok {
		t.Fatalf("second message type = %T, want SysStatus", msg)
	}
}

func TestUDPSourceReadsMAVLinkV1Frame(t *testing.T) {
	source, err := ListenUDP(UDPSourceConfig{
		ListenAddr:    "127.0.0.1:0",
		SubjectPrefix: "mavlink.raw.ardupilot-sitl",
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

	frame, err := EncodeV1(7, 42, 1, MessageHeartbeat, HeartbeatPayloadForType(false, 0, MavTypeGroundRover))
	if err != nil {
		t.Fatalf("EncodeV1() error = %v", err)
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
	msg, err := DecodeMessage(got.Bytes)
	if err != nil {
		t.Fatalf("DecodeMessage() error = %v", err)
	}
	if _, ok := msg.(Heartbeat); !ok {
		t.Fatalf("message type = %T, want Heartbeat", msg)
	}
}

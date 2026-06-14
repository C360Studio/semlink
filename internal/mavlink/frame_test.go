package mavlink

import "testing"

func TestEncodeDecodeGlobalPositionInt(t *testing.T) {
	payload := GlobalPositionIntPayload(42, 388895000, -770353000, 123000, 78000, 120, 30, -4, 9010)
	frame, err := EncodeV2(7, 3, 1, MessageGlobalPositionInt, payload)
	if err != nil {
		t.Fatalf("EncodeV2: %v", err)
	}

	msg, err := DecodeMessage(frame)
	if err != nil {
		t.Fatalf("DecodeMessage: %v", err)
	}
	got, ok := msg.(GlobalPositionInt)
	if !ok {
		t.Fatalf("message type = %T, want GlobalPositionInt", msg)
	}
	if got.System() != 3 || got.SequenceNumber() != 7 {
		t.Fatalf("envelope = sys %d seq %d", got.System(), got.SequenceNumber())
	}
	if got.LatitudeDeg() != 38.8895 || got.LongitudeDeg() != -77.0353 {
		t.Fatalf("position = %f,%f", got.LatitudeDeg(), got.LongitudeDeg())
	}
}

func TestDecodeRejectsCorruptChecksum(t *testing.T) {
	frame, err := EncodeV2(1, 1, 1, MessageHeartbeat, HeartbeatPayload(true, 0))
	if err != nil {
		t.Fatalf("EncodeV2: %v", err)
	}
	frame[len(frame)-1] ^= 0xff

	if _, err := DecodeMessage(frame); err == nil {
		t.Fatal("DecodeMessage accepted corrupt checksum")
	}
}

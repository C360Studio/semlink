package cot

import (
	"strings"
	"testing"
	"time"
)

func TestMarshalAndUnmarshalTrack(t *testing.T) {
	now := time.Date(2026, 6, 16, 12, 30, 0, 0, time.UTC)
	raw, err := Marshal(Event{
		UID:       "uav-001",
		Type:      TypeAirTrack,
		Time:      now,
		Point:     &Point{Lat: 38.8895, Lon: -77.0353, HAE: 120, CE: 5, LE: 10},
		Callsign:  "UAV-001",
		CourseDeg: 90,
		SpeedMPS:  6.2,
		HasTrack:  true,
	})
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	text := string(raw)
	for _, want := range []string{`uid="uav-001"`, `type="a-f-A-M-F-Q"`, `<point`, `<contact callsign="UAV-001"`, `<track course="90" speed="6.2"`} {
		if !strings.Contains(text, want) {
			t.Fatalf("encoded CoT missing %q:\n%s", want, text)
		}
	}

	got, err := Unmarshal(raw)
	if err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if got.UID != "uav-001" || got.Type != TypeAirTrack || got.Point == nil {
		t.Fatalf("decoded event = %#v", got)
	}
	if got.Callsign != "UAV-001" || !got.HasTrack || got.SpeedMPS != 6.2 {
		t.Fatalf("decoded detail = %#v", got)
	}
}

func TestUnmarshalGeoChatUsesRemarksAsText(t *testing.T) {
	raw := []byte(`<event version="2.0" uid="chat-1" type="b-t-f" how="h-g-i-g-o" time="2026-06-16T12:30:00Z"><detail><contact callsign="ALPHA"/><remarks>hold at checkpoint</remarks><__chat senderUid="ANDROID-ALPHA"/></detail></event>`)
	got, err := Unmarshal(raw)
	if err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if got.Callsign != "ALPHA" || got.ChatText != "hold at checkpoint" || got.SenderUID != "ANDROID-ALPHA" {
		t.Fatalf("decoded geochat = %#v", got)
	}
}

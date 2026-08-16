package cop

import (
	"strings"
	"testing"
	"time"

	"github.com/c360studio/semlink/internal/cot"
	"github.com/c360studio/semstreams/pkg/projection"
	"github.com/c360studio/semstreams/vocabulary"
)

func TestContractsValidateAndDeclareProfiles(t *testing.T) {
	contracts := Contracts()
	if err := projection.ValidateContracts(contracts); err != nil {
		t.Fatalf("ValidateContracts: %v", err)
	}
	profiles := map[string]string{}
	for _, contract := range contracts {
		profiles[contract.EntityPattern] = contract.IndexingProfile
	}
	if profiles["c360.semlink.cop.operator.position.*"] != vocabulary.IndexingProfileSignal {
		t.Fatalf("operator profile = %q", profiles["c360.semlink.cop.operator.position.*"])
	}
	if profiles["c360.semlink.cop.marker.poi.*"] != vocabulary.IndexingProfileContent {
		t.Fatalf("marker profile = %q", profiles["c360.semlink.cop.marker.poi.*"])
	}
	if profiles["c360.semlink.cop.message.geochat.*"] != vocabulary.IndexingProfileContent {
		t.Fatalf("message profile = %q", profiles["c360.semlink.cop.message.geochat.*"])
	}
	for _, contract := range contracts {
		if len(contract.Groups) != 1 || contract.Groups[0].Name == "" || contract.Groups[0].Mode != projection.ModeReconcile {
			t.Fatalf("contract %q groups = %#v", contract.Name, contract.Groups)
		}
	}
}

func TestEntityIDIsCollisionSafeForDistinctUIDs(t *testing.T) {
	first, err := EntityID(KindOperator, "alpha/beta")
	if err != nil {
		t.Fatalf("EntityID first: %v", err)
	}
	second, err := EntityID(KindOperator, "alpha beta")
	if err != nil {
		t.Fatalf("EntityID second: %v", err)
	}
	if first == second {
		t.Fatalf("distinct CoT UIDs collapsed to one entity id: %q", first)
	}
	if strings.Contains(first, "/") || strings.Contains(second, " ") {
		t.Fatalf("entity ids are not token-safe: %q %q", first, second)
	}
}

func TestTranslatorProjectsOperatorMarkerAndMessage(t *testing.T) {
	tr := NewTranslator()
	now := time.Date(2026, 6, 16, 12, 30, 0, 0, time.UTC)

	operator, ok, err := tr.Apply(cot.Event{
		UID:       "ANDROID-1",
		Type:      cot.TypeOperatorPosition,
		Time:      now,
		Point:     &cot.Point{Lat: 38.9, Lon: -77.03, HAE: 22},
		Callsign:  "ALPHA",
		CourseDeg: 180,
		SpeedMPS:  1.5,
		HasTrack:  true,
	}, now)
	if err != nil || !ok {
		t.Fatalf("operator Apply ok=%t err=%v", ok, err)
	}
	if operator.Projection.IndexingProfile != vocabulary.IndexingProfileSignal || operator.View.Kind != KindOperator {
		t.Fatalf("operator result = %#v", operator)
	}
	if operator.Projection.Contract != OperatorType.String() || operator.Projection.Group != OperatorGroup {
		t.Fatalf("operator binding = %q/%q", operator.Projection.Contract, operator.Projection.Group)
	}

	marker, ok, err := tr.Apply(cot.Event{
		UID:      "marker-1",
		Type:     cot.TypeMarker,
		Time:     now,
		Point:    &cot.Point{Lat: 39, Lon: -77, HAE: 0},
		Callsign: "Checkpoint",
		Remarks:  "north gate",
	}, now)
	if err != nil || !ok {
		t.Fatalf("marker Apply ok=%t err=%v", ok, err)
	}
	if marker.Projection.IndexingProfile != vocabulary.IndexingProfileContent || marker.View.Kind != KindMarker {
		t.Fatalf("marker result = %#v", marker)
	}

	message, ok, err := tr.Apply(cot.Event{
		UID:       "chat-1",
		Type:      cot.TypeGeoChat,
		Time:      now,
		Callsign:  "ALPHA",
		SenderUID: "ANDROID-1",
		ChatText:  "hold at checkpoint",
	}, now)
	if err != nil || !ok {
		t.Fatalf("message Apply ok=%t err=%v", ok, err)
	}
	if message.Projection.IndexingProfile != vocabulary.IndexingProfileContent || message.View.Text != "hold at checkpoint" {
		t.Fatalf("message result = %#v", message)
	}
	if message.View.SenderUID != "ANDROID-1" || message.View.SenderEntity != operator.View.EntityID {
		t.Fatalf("message sender = %q/%q, want %q/%q", message.View.SenderUID, message.View.SenderEntity, "ANDROID-1", operator.View.EntityID)
	}
}

func TestTranslatorIgnoresInboundUAVTracks(t *testing.T) {
	tr := NewTranslator()
	_, ok, err := tr.Apply(cot.Event{
		UID:   "uav-001",
		Type:  cot.TypeAirTrack,
		Point: &cot.Point{Lat: 38.9, Lon: -77.03},
	}, time.Now())
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if ok {
		t.Fatal("inbound UAV track should stay out of Phase 1")
	}
}

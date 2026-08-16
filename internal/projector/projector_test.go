package projector

import (
	"testing"
	"time"

	"github.com/c360studio/semlink/internal/mavlink"
	"github.com/c360studio/semstreams/pkg/projection"
	"github.com/c360studio/semstreams/vocabulary"
)

func TestContractsValidateAndDeclareProfiles(t *testing.T) {
	contracts := Contracts()
	if err := projection.ValidateContracts(contracts); err != nil {
		t.Fatalf("ValidateContracts: %v", err)
	}
	if contracts[0].IndexingProfile != vocabulary.IndexingProfileSignal {
		t.Fatalf("vehicle profile = %q", contracts[0].IndexingProfile)
	}
	if contracts[1].IndexingProfile != vocabulary.IndexingProfileControl {
		t.Fatalf("alert profile = %q", contracts[1].IndexingProfile)
	}
	for _, contract := range contracts {
		if len(contract.Groups) != 1 || contract.Groups[0].Name == "" || contract.Groups[0].Mode != projection.ModeReconcile {
			t.Fatalf("contract %q groups = %#v", contract.Name, contract.Groups)
		}
	}
}

func TestProjectorCollapsesRawMessagesToCurrentVehicleEntity(t *testing.T) {
	p := New(DefaultConfig())
	now := time.Unix(100, 0)

	for i := 0; i < 10; i++ {
		frame, err := mavlink.EncodeV2(uint8(i), 1, 1, mavlink.MessageGlobalPositionInt,
			mavlink.GlobalPositionIntPayload(uint32(i), 388895000+int32(i), -770353000, 100000, 50000, 100, 0, 0, 9000))
		if err != nil {
			t.Fatalf("EncodeV2: %v", err)
		}
		msg, err := mavlink.DecodeMessage(frame)
		if err != nil {
			t.Fatalf("DecodeMessage: %v", err)
		}
		projections := p.Apply(msg, now.Add(time.Duration(i)*time.Millisecond))
		if len(projections) != 1 {
			t.Fatalf("projection count = %d", len(projections))
		}
		if projections[0].Entity.ID != VehicleEntityID(1) {
			t.Fatalf("entity id = %q", projections[0].Entity.ID)
		}
		if projections[0].IndexingProfile != vocabulary.IndexingProfileSignal {
			t.Fatalf("profile = %q", projections[0].IndexingProfile)
		}
		if projections[0].Contract != VehicleTelemetryType.String() || projections[0].Group != VehicleTelemetryGroup {
			t.Fatalf("binding = %q/%q", projections[0].Contract, projections[0].Group)
		}
	}

	vehicles, _ := p.Snapshot()
	if len(vehicles) != 1 {
		t.Fatalf("vehicle count = %d", len(vehicles))
	}
	if vehicles[0].Sequence != 9 {
		t.Fatalf("sequence = %d", vehicles[0].Sequence)
	}
}

func TestProjectorUsesHeartbeatMAVTypeForVehicleType(t *testing.T) {
	p := New(DefaultConfig())
	now := time.Unix(150, 0)
	frame, err := mavlink.EncodeV2(1, 3, 1, mavlink.MessageHeartbeat,
		mavlink.HeartbeatPayloadForType(true, 0, mavlink.MavTypeGroundRover))
	if err != nil {
		t.Fatalf("EncodeV2: %v", err)
	}
	msg, err := mavlink.DecodeMessage(frame)
	if err != nil {
		t.Fatalf("DecodeMessage: %v", err)
	}

	projections := p.Apply(msg, now)
	if len(projections) != 1 {
		t.Fatalf("projection count = %d", len(projections))
	}

	vehicles, _ := p.Snapshot()
	if len(vehicles) != 1 {
		t.Fatalf("vehicle count = %d", len(vehicles))
	}
	if vehicles[0].VehicleType != "ground-rover" {
		t.Fatalf("vehicle type = %q, want ground-rover", vehicles[0].VehicleType)
	}

	var gotType any
	for _, triple := range projections[0].Triples {
		if triple.Predicate == PredicateVehicleType {
			gotType = triple.Object
			break
		}
	}
	if gotType == nil {
		t.Fatal("vehicle type triple missing")
	}
	if gotType != "ground-rover" {
		t.Fatalf("vehicle type triple = %v, want ground-rover", gotType)
	}
}

func TestProjectorEmitsControlPlaneAlerts(t *testing.T) {
	p := New(Config{LowBatteryThreshold: 25, LostLinkAfter: time.Second})
	now := time.Unix(200, 0)
	status, err := mavlink.EncodeV2(1, 2, 1, mavlink.MessageSysStatus, mavlink.SysStatusPayload(10900, -1, 18, 0))
	if err != nil {
		t.Fatalf("EncodeV2: %v", err)
	}
	msg, err := mavlink.DecodeMessage(status)
	if err != nil {
		t.Fatalf("DecodeMessage: %v", err)
	}

	projections := p.Apply(msg, now)
	if len(projections) != 2 {
		t.Fatalf("projection count = %d", len(projections))
	}
	if projections[1].IndexingProfile != vocabulary.IndexingProfileControl {
		t.Fatalf("alert profile = %q", projections[1].IndexingProfile)
	}

	lost := p.CheckLinkTimeouts(now.Add(2 * time.Second))
	if len(lost) == 0 {
		t.Fatal("expected lost-link projections")
	}
}

package mavlink

import (
	"fmt"
	"math"
	"time"
)

// RawFrame is one MAVLink frame emitted by a source adapter.
type RawFrame struct {
	Subject   string
	VehicleID string
	SystemID  uint8
	EmittedAt time.Time
	Bytes     []byte
}

// Simulator emits valid MAVLink v2 frames for a deterministic vehicle swarm.
type Simulator struct {
	vehicles int
	start    time.Time
	seq      uint8
}

func NewSimulator(vehicles int) *Simulator {
	if vehicles < 1 {
		vehicles = 1
	}
	return &Simulator{vehicles: vehicles, start: time.Now()}
}

// Next returns the frames due for the given tick. The last vehicle periodically
// drops all frames to exercise lost-link detection without faking the projector.
func (s *Simulator) Next(now time.Time) ([]RawFrame, error) {
	elapsed := now.Sub(s.start)
	phase := elapsed.Seconds()
	out := make([]RawFrame, 0, s.vehicles*3)

	for i := 1; i <= s.vehicles; i++ {
		systemID := uint8(i)
		vehicleID := fmt.Sprintf("uav-%03d", i)
		if i == s.vehicles && int(phase)%24 >= 14 && int(phase)%24 <= 19 {
			continue
		}

		angle := phase*0.18 + float64(i)*0.75
		lat := 38.8895 + math.Sin(angle)*0.008 + float64(i)*0.0012
		lon := -77.0353 + math.Cos(angle)*0.011 - float64(i)*0.001
		alt := 80 + math.Sin(angle*1.7)*18 + float64(i)*6
		speed := 550 + int16(80*math.Sin(angle))
		heading := uint16(math.Mod(angle*1800/math.Pi+36000, 36000))
		battery := int8(92 - int(elapsed.Seconds()/5) - i*4)
		if i == 1 && elapsed > 18*time.Second {
			battery = 18
		}
		if battery < 7 {
			battery = 7
		}

		heartbeat, err := s.frame(systemID, MessageHeartbeat, HeartbeatPayload(true, 0))
		if err != nil {
			return nil, err
		}
		position, err := s.frame(systemID, MessageGlobalPositionInt, GlobalPositionIntPayload(
			uint32(elapsed.Milliseconds()),
			int32(lat*1e7),
			int32(lon*1e7),
			int32(alt*1000),
			int32((alt-45)*1000),
			speed,
			int16(speed/3),
			0,
			heading,
		))
		if err != nil {
			return nil, err
		}
		status, err := s.frame(systemID, MessageSysStatus, SysStatusPayload(uint16(11800+int(battery)*10), -1, battery, 0))
		if err != nil {
			return nil, err
		}

		subject := "mavlink.raw." + vehicleID
		out = append(out,
			RawFrame{Subject: subject, VehicleID: vehicleID, SystemID: systemID, EmittedAt: now, Bytes: heartbeat},
			RawFrame{Subject: subject, VehicleID: vehicleID, SystemID: systemID, EmittedAt: now, Bytes: position},
			RawFrame{Subject: subject, VehicleID: vehicleID, SystemID: systemID, EmittedAt: now, Bytes: status},
		)
	}
	return out, nil
}

func (s *Simulator) frame(systemID uint8, messageID MessageID, payload []byte) ([]byte, error) {
	frame, err := EncodeV2(s.seq, systemID, 1, messageID, payload)
	s.seq++
	return frame, err
}

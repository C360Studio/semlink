package mavlink

import (
	"encoding/binary"
	"fmt"
	"math"
)

const (
	MavTypeQuadrotor       = 2
	MavTypeGroundRover     = 10
	MavTypeSurfaceBoat     = 11
	MavAutopilotGeneric    = 0
	MavModeFlagSafetyArmed = 0x80
	MavStateActive         = 4

	MAVCmdRequestMessage = 512

	MAVResultAccepted            = 0
	MAVResultTemporarilyRejected = 1
	MAVResultDenied              = 2
	MAVResultUnsupported         = 3
	MAVResultFailed              = 4
)

// MAVTypeName returns the graph-facing vehicle type label for the MAVLink
// heartbeat MAV_TYPE values SemLink currently projects.
func MAVTypeName(vehicleType uint8) string {
	switch vehicleType {
	case MavTypeQuadrotor:
		return "quadrotor"
	case MavTypeGroundRover:
		return "ground-rover"
	case MavTypeSurfaceBoat:
		return "surface-boat"
	default:
		return fmt.Sprintf("mav-type-%d", vehicleType)
	}
}

// Message is a decoded MAVLink message supported by this demo.
type Message interface {
	ID() MessageID
	System() uint8
	Component() uint8
	SequenceNumber() uint8
}

type envelope struct {
	id        MessageID
	system    uint8
	component uint8
	sequence  uint8
}

func (e envelope) ID() MessageID         { return e.id }
func (e envelope) System() uint8         { return e.system }
func (e envelope) Component() uint8      { return e.component }
func (e envelope) SequenceNumber() uint8 { return e.sequence }

// Heartbeat is MAVLink common HEARTBEAT (#0).
type Heartbeat struct {
	envelope
	CustomMode   uint32
	Type         uint8
	Autopilot    uint8
	BaseMode     uint8
	SystemStatus uint8
	MAVLink      uint8
}

// Armed reports whether MAV_MODE_FLAG_SAFETY_ARMED is set.
func (h Heartbeat) Armed() bool {
	return h.BaseMode&MavModeFlagSafetyArmed != 0
}

// SysStatus is MAVLink common SYS_STATUS (#1), narrowed to the fields the
// control-plane demo needs.
type SysStatus struct {
	envelope
	Load             uint16
	VoltageBatteryMV uint16
	CurrentBatteryCA int16
	BatteryRemaining int8
	DropRateComm     uint16
	ErrorsComm       uint16
}

// GlobalPositionInt is MAVLink common GLOBAL_POSITION_INT (#33).
type GlobalPositionInt struct {
	envelope
	TimeBootMS    uint32
	LatitudeE7    int32
	LongitudeE7   int32
	AltitudeMM    int32
	RelativeAltMM int32
	VXCMS         int16
	VYCMS         int16
	VZCMS         int16
	HeadingCDeg   uint16
}

func (g GlobalPositionInt) LatitudeDeg() float64  { return float64(g.LatitudeE7) / 1e7 }
func (g GlobalPositionInt) LongitudeDeg() float64 { return float64(g.LongitudeE7) / 1e7 }
func (g GlobalPositionInt) AltitudeM() float64    { return float64(g.AltitudeMM) / 1000 }
func (g GlobalPositionInt) GroundSpeedMS() float64 {
	return math.Hypot(float64(g.VXCMS), float64(g.VYCMS)) / 100
}

type CommandLongFields struct {
	Params          [7]float32
	Command         uint16
	TargetSystem    uint8
	TargetComponent uint8
	Confirmation    uint8
}

// CommandAck is MAVLink common COMMAND_ACK (#77).
type CommandAck struct {
	envelope
	Command         uint16
	Result          uint8
	Progress        uint8
	ResultParam2    int32
	TargetSystem    uint8
	TargetComponent uint8
}

func (a CommandAck) Accepted() bool {
	return a.Result == MAVResultAccepted
}

// DecodeMessage decodes a supported MAVLink frame payload into a typed message.
func DecodeMessage(data []byte) (Message, error) {
	frame, err := DecodeFrame(data)
	if err != nil {
		return nil, err
	}
	env := envelope{
		id:        frame.MessageID,
		system:    frame.SystemID,
		component: frame.ComponentID,
		sequence:  frame.Sequence,
	}

	switch frame.MessageID {
	case MessageHeartbeat:
		if len(frame.Payload) < 9 {
			return nil, fmt.Errorf("mavlink: HEARTBEAT payload too short: %d", len(frame.Payload))
		}
		return Heartbeat{
			envelope:     env,
			CustomMode:   binary.LittleEndian.Uint32(frame.Payload[0:4]),
			Type:         frame.Payload[4],
			Autopilot:    frame.Payload[5],
			BaseMode:     frame.Payload[6],
			SystemStatus: frame.Payload[7],
			MAVLink:      frame.Payload[8],
		}, nil
	case MessageSysStatus:
		if len(frame.Payload) < 31 {
			return nil, fmt.Errorf("mavlink: SYS_STATUS payload too short: %d", len(frame.Payload))
		}
		return SysStatus{
			envelope:         env,
			Load:             binary.LittleEndian.Uint16(frame.Payload[12:14]),
			VoltageBatteryMV: binary.LittleEndian.Uint16(frame.Payload[14:16]),
			CurrentBatteryCA: int16(binary.LittleEndian.Uint16(frame.Payload[16:18])),
			DropRateComm:     binary.LittleEndian.Uint16(frame.Payload[18:20]),
			ErrorsComm:       binary.LittleEndian.Uint16(frame.Payload[20:22]),
			BatteryRemaining: int8(frame.Payload[30]),
		}, nil
	case MessageGlobalPositionInt:
		if len(frame.Payload) < 28 {
			return nil, fmt.Errorf("mavlink: GLOBAL_POSITION_INT payload too short: %d", len(frame.Payload))
		}
		return GlobalPositionInt{
			envelope:      env,
			TimeBootMS:    binary.LittleEndian.Uint32(frame.Payload[0:4]),
			LatitudeE7:    int32(binary.LittleEndian.Uint32(frame.Payload[4:8])),
			LongitudeE7:   int32(binary.LittleEndian.Uint32(frame.Payload[8:12])),
			AltitudeMM:    int32(binary.LittleEndian.Uint32(frame.Payload[12:16])),
			RelativeAltMM: int32(binary.LittleEndian.Uint32(frame.Payload[16:20])),
			VXCMS:         int16(binary.LittleEndian.Uint16(frame.Payload[20:22])),
			VYCMS:         int16(binary.LittleEndian.Uint16(frame.Payload[22:24])),
			VZCMS:         int16(binary.LittleEndian.Uint16(frame.Payload[24:26])),
			HeadingCDeg:   binary.LittleEndian.Uint16(frame.Payload[26:28]),
		}, nil
	case MessageCommandAck:
		if len(frame.Payload) < 3 {
			return nil, fmt.Errorf("mavlink: COMMAND_ACK payload too short: %d", len(frame.Payload))
		}
		ack := CommandAck{
			envelope: env,
			Command:  binary.LittleEndian.Uint16(frame.Payload[0:2]),
			Result:   frame.Payload[2],
		}
		if len(frame.Payload) >= 10 {
			ack.Progress = frame.Payload[3]
			ack.ResultParam2 = int32(binary.LittleEndian.Uint32(frame.Payload[4:8]))
			ack.TargetSystem = frame.Payload[8]
			ack.TargetComponent = frame.Payload[9]
		}
		return ack, nil
	default:
		return nil, fmt.Errorf("mavlink: unsupported decoded message id %d", frame.MessageID)
	}
}

func HeartbeatPayload(armed bool, customMode uint32) []byte {
	return HeartbeatPayloadForType(armed, customMode, MavTypeQuadrotor)
}

func HeartbeatPayloadForType(armed bool, customMode uint32, vehicleType uint8) []byte {
	payload := make([]byte, 9)
	binary.LittleEndian.PutUint32(payload[0:4], customMode)
	payload[4] = vehicleType
	payload[5] = MavAutopilotGeneric
	if armed {
		payload[6] = MavModeFlagSafetyArmed
	}
	payload[7] = MavStateActive
	payload[8] = 3
	return payload
}

func SysStatusPayload(voltageMV uint16, currentCA int16, batteryRemaining int8, dropRate uint16) []byte {
	payload := make([]byte, 31)
	binary.LittleEndian.PutUint16(payload[12:14], 500)
	binary.LittleEndian.PutUint16(payload[14:16], voltageMV)
	binary.LittleEndian.PutUint16(payload[16:18], uint16(currentCA))
	binary.LittleEndian.PutUint16(payload[18:20], dropRate)
	payload[30] = byte(batteryRemaining)
	return payload
}

func GlobalPositionIntPayload(timeBootMS uint32, latE7, lonE7, altMM, relAltMM int32, vxCMS, vyCMS, vzCMS int16, headingCDeg uint16) []byte {
	payload := make([]byte, 28)
	binary.LittleEndian.PutUint32(payload[0:4], timeBootMS)
	binary.LittleEndian.PutUint32(payload[4:8], uint32(latE7))
	binary.LittleEndian.PutUint32(payload[8:12], uint32(lonE7))
	binary.LittleEndian.PutUint32(payload[12:16], uint32(altMM))
	binary.LittleEndian.PutUint32(payload[16:20], uint32(relAltMM))
	binary.LittleEndian.PutUint16(payload[20:22], uint16(vxCMS))
	binary.LittleEndian.PutUint16(payload[22:24], uint16(vyCMS))
	binary.LittleEndian.PutUint16(payload[24:26], uint16(vzCMS))
	binary.LittleEndian.PutUint16(payload[26:28], headingCDeg)
	return payload
}

func RequestMessageCommandPayload(targetSystem, targetComponent uint8, messageID MessageID, confirmation uint8) []byte {
	var params [7]float32
	params[0] = float32(messageID)
	return CommandLongPayload(targetSystem, targetComponent, MAVCmdRequestMessage, confirmation, params)
}

func CommandLongPayload(targetSystem, targetComponent uint8, command uint16, confirmation uint8, params [7]float32) []byte {
	payload := make([]byte, 33)
	for i, param := range params {
		start := i * 4
		binary.LittleEndian.PutUint32(payload[start:start+4], math.Float32bits(param))
	}
	binary.LittleEndian.PutUint16(payload[28:30], command)
	payload[30] = targetSystem
	payload[31] = targetComponent
	payload[32] = confirmation
	return payload
}

func DecodeCommandLongPayload(payload []byte) (CommandLongFields, error) {
	if len(payload) < 33 {
		return CommandLongFields{}, fmt.Errorf("mavlink: COMMAND_LONG payload too short: %d", len(payload))
	}
	var params [7]float32
	for i := range params {
		start := i * 4
		params[i] = math.Float32frombits(binary.LittleEndian.Uint32(payload[start : start+4]))
	}
	return CommandLongFields{
		Params:          params,
		Command:         binary.LittleEndian.Uint16(payload[28:30]),
		TargetSystem:    payload[30],
		TargetComponent: payload[31],
		Confirmation:    payload[32],
	}, nil
}

func CommandAckPayload(command uint16, result, progress uint8, resultParam2 int32, targetSystem, targetComponent uint8) []byte {
	payload := make([]byte, 10)
	binary.LittleEndian.PutUint16(payload[0:2], command)
	payload[2] = result
	payload[3] = progress
	binary.LittleEndian.PutUint32(payload[4:8], uint32(resultParam2))
	payload[8] = targetSystem
	payload[9] = targetComponent
	return payload
}

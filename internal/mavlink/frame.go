package mavlink

import (
	"encoding/binary"
	"fmt"
)

// Frame is a validated MAVLink frame.
type Frame struct {
	Sequence    uint8
	SystemID    uint8
	ComponentID uint8
	MessageID   MessageID
	Payload     []byte
}

// EncodeV1 creates a MAVLink 1 frame for a supported message id.
func EncodeV1(seq, systemID, componentID uint8, messageID MessageID, payload []byte) ([]byte, error) {
	if len(payload) > 255 {
		return nil, fmt.Errorf("mavlink: payload too large: %d", len(payload))
	}
	if messageID > 255 {
		return nil, fmt.Errorf("mavlink: message id %d cannot be encoded as MAVLink 1", messageID)
	}
	extra, err := messageCRCExtra(messageID)
	if err != nil {
		return nil, err
	}

	frame := make([]byte, headerLenV1+len(payload)+checksumLen)
	frame[0] = stxV1
	frame[1] = byte(len(payload))
	frame[2] = seq
	frame[3] = systemID
	frame[4] = componentID
	frame[5] = byte(messageID)
	copy(frame[headerLenV1:], payload)

	crc := x25Checksum(frame[1:headerLenV1+len(payload)], extra)
	binary.LittleEndian.PutUint16(frame[headerLenV1+len(payload):], crc)
	return frame, nil
}

// EncodeV2 creates an unsigned MAVLink 2 frame for a supported message id.
func EncodeV2(seq, systemID, componentID uint8, messageID MessageID, payload []byte) ([]byte, error) {
	if len(payload) > 255 {
		return nil, fmt.Errorf("mavlink: payload too large: %d", len(payload))
	}
	extra, err := messageCRCExtra(messageID)
	if err != nil {
		return nil, err
	}

	frame := make([]byte, headerLenV2+len(payload)+checksumLen)
	frame[0] = stxV2
	frame[1] = byte(len(payload))
	frame[2] = 0
	frame[3] = 0
	frame[4] = seq
	frame[5] = systemID
	frame[6] = componentID
	frame[7] = byte(messageID)
	frame[8] = byte(messageID >> 8)
	frame[9] = byte(messageID >> 16)
	copy(frame[headerLenV2:], payload)

	crc := x25Checksum(frame[1:headerLenV2+len(payload)], extra)
	binary.LittleEndian.PutUint16(frame[headerLenV2+len(payload):], crc)
	return frame, nil
}

// DecodeFrame validates a supported MAVLink 1 or unsigned MAVLink 2 frame and
// returns its envelope.
func DecodeFrame(data []byte) (*Frame, error) {
	if len(data) < headerLenV1+checksumLen {
		return nil, fmt.Errorf("mavlink: frame too short: %d", len(data))
	}
	switch data[0] {
	case stxV1:
		return decodeFrameV1(data)
	case stxV2:
		return decodeFrameV2(data)
	default:
		return nil, fmt.Errorf("mavlink: unsupported magic 0x%x", data[0])
	}
}

func decodeFrameV1(data []byte) (*Frame, error) {
	payloadLen := int(data[1])
	wantLen := headerLenV1 + payloadLen + checksumLen
	if len(data) != wantLen {
		return nil, fmt.Errorf("mavlink: length mismatch got %d want %d", len(data), wantLen)
	}

	messageID := MessageID(data[5])
	extra, err := messageCRCExtra(messageID)
	if err != nil {
		return nil, err
	}
	payloadEnd := headerLenV1 + payloadLen
	gotCRC := binary.LittleEndian.Uint16(data[payloadEnd : payloadEnd+checksumLen])
	wantCRC := x25Checksum(data[1:payloadEnd], extra)
	if gotCRC != wantCRC {
		return nil, fmt.Errorf("mavlink: checksum mismatch got 0x%x want 0x%x", gotCRC, wantCRC)
	}

	payload := make([]byte, payloadLen)
	copy(payload, data[headerLenV1:payloadEnd])
	return &Frame{
		Sequence:    data[2],
		SystemID:    data[3],
		ComponentID: data[4],
		MessageID:   messageID,
		Payload:     payload,
	}, nil
}

func decodeFrameV2(data []byte) (*Frame, error) {
	if len(data) < headerLenV2+checksumLen {
		return nil, fmt.Errorf("mavlink: frame too short: %d", len(data))
	}
	payloadLen := int(data[1])
	signed := data[2]&incompatFlagSigned != 0
	wantLen := headerLenV2 + payloadLen + checksumLen
	if signed {
		wantLen += signatureLen
	}
	if len(data) != wantLen {
		return nil, fmt.Errorf("mavlink: length mismatch got %d want %d", len(data), wantLen)
	}
	if signed {
		return nil, fmt.Errorf("mavlink: signed frames are not supported by this demo decoder")
	}

	messageID := MessageID(uint32(data[7]) | uint32(data[8])<<8 | uint32(data[9])<<16)
	extra, err := messageCRCExtra(messageID)
	if err != nil {
		return nil, err
	}
	payloadEnd := headerLenV2 + payloadLen
	gotCRC := binary.LittleEndian.Uint16(data[payloadEnd : payloadEnd+checksumLen])
	wantCRC := x25Checksum(data[1:payloadEnd], extra)
	if gotCRC != wantCRC {
		return nil, fmt.Errorf("mavlink: checksum mismatch got 0x%x want 0x%x", gotCRC, wantCRC)
	}

	payload := make([]byte, payloadLen)
	copy(payload, data[headerLenV2:payloadEnd])
	return &Frame{
		Sequence:    data[4],
		SystemID:    data[5],
		ComponentID: data[6],
		MessageID:   messageID,
		Payload:     payload,
	}, nil
}

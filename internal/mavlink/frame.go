package mavlink

import (
	"encoding/binary"
	"fmt"
)

// Frame is a validated MAVLink 2 frame without the optional signature.
type Frame struct {
	Sequence    uint8
	SystemID    uint8
	ComponentID uint8
	MessageID   MessageID
	Payload     []byte
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

// DecodeFrame validates an unsigned MAVLink 2 frame and returns its envelope.
func DecodeFrame(data []byte) (*Frame, error) {
	if len(data) < headerLenV2+checksumLen {
		return nil, fmt.Errorf("mavlink: frame too short: %d", len(data))
	}
	if data[0] != stxV2 {
		return nil, fmt.Errorf("mavlink: unsupported magic 0x%x", data[0])
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

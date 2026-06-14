package mavlink

import "fmt"

// MessageID is the MAVLink common.xml message identifier.
type MessageID uint32

const (
	MessageHeartbeat         MessageID = 0
	MessageSysStatus         MessageID = 1
	MessageGlobalPositionInt MessageID = 33
)

const (
	stxV2              = 0xfd
	headerLenV2        = 10
	checksumLen        = 2
	signatureLen       = 13
	incompatFlagSigned = 0x01
)

var crcExtra = map[MessageID]byte{
	MessageHeartbeat:         50,
	MessageSysStatus:         124,
	MessageGlobalPositionInt: 104,
}

func messageCRCExtra(id MessageID) (byte, error) {
	extra, ok := crcExtra[id]
	if !ok {
		return 0, fmt.Errorf("mavlink: unsupported message id %d", id)
	}
	return extra, nil
}

func x25Checksum(data []byte, extra byte) uint16 {
	crc := uint16(0xffff)
	for _, b := range data {
		tmp := byte(b) ^ byte(crc&0xff)
		tmp ^= tmp << 4
		crc = (crc >> 8) ^ (uint16(tmp) << 8) ^ (uint16(tmp) << 3) ^ (uint16(tmp) >> 4)
	}
	tmp := extra ^ byte(crc&0xff)
	tmp ^= tmp << 4
	return (crc >> 8) ^ (uint16(tmp) << 8) ^ (uint16(tmp) << 3) ^ (uint16(tmp) >> 4)
}

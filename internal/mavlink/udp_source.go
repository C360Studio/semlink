package mavlink

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strings"
	"time"
)

type Clock func() time.Time

type UDPSourceConfig struct {
	ListenAddr       string
	SubjectPrefix    string
	Clock            Clock
	ReadDeadline     time.Duration
	MaxDatagramBytes int
}

type UDPSource struct {
	conn             *net.UDPConn
	subjectPrefix    string
	clock            Clock
	readDeadline     time.Duration
	maxDatagramBytes int
	pending          []RawFrame
}

func ListenUDP(cfg UDPSourceConfig) (*UDPSource, error) {
	listenAddr := cfg.ListenAddr
	if listenAddr == "" {
		listenAddr = "127.0.0.1:14550"
	}
	udpAddr, err := net.ResolveUDPAddr("udp", listenAddr)
	if err != nil {
		return nil, fmt.Errorf("resolve MAVLink UDP listen address: %w", err)
	}
	network := "udp4"
	if udpAddr.IP != nil && udpAddr.IP.To4() == nil {
		network = "udp6"
	}
	conn, err := net.ListenUDP(network, udpAddr)
	if err != nil {
		return nil, fmt.Errorf("listen for MAVLink UDP frames: %w", err)
	}

	subjectPrefix := strings.TrimSuffix(cfg.SubjectPrefix, ".")
	if subjectPrefix == "" {
		subjectPrefix = "mavlink.raw.sitl"
	}
	clock := cfg.Clock
	if clock == nil {
		clock = time.Now
	}
	readDeadline := cfg.ReadDeadline
	if readDeadline <= 0 {
		readDeadline = 100 * time.Millisecond
	}
	maxDatagramBytes := cfg.MaxDatagramBytes
	if maxDatagramBytes <= 0 {
		maxDatagramBytes = 4096
	}

	return &UDPSource{
		conn:             conn,
		subjectPrefix:    subjectPrefix,
		clock:            clock,
		readDeadline:     readDeadline,
		maxDatagramBytes: maxDatagramBytes,
	}, nil
}

func (s *UDPSource) LocalAddr() net.Addr {
	return s.conn.LocalAddr()
}

func (s *UDPSource) Close() error {
	return s.conn.Close()
}

func (s *UDPSource) Read(ctx context.Context) (RawFrame, error) {
	buf := make([]byte, s.maxDatagramBytes)
	for {
		if len(s.pending) > 0 {
			frame := s.pending[0]
			s.pending = s.pending[1:]
			return frame, nil
		}
		if err := ctx.Err(); err != nil {
			return RawFrame{}, err
		}
		if err := s.conn.SetReadDeadline(time.Now().Add(s.readDeadline)); err != nil {
			return RawFrame{}, fmt.Errorf("set MAVLink UDP read deadline: %w", err)
		}
		n, _, err := s.conn.ReadFromUDP(buf)
		if err != nil {
			if isTimeout(err) {
				continue
			}
			if errors.Is(err, net.ErrClosed) {
				return RawFrame{}, err
			}
			return RawFrame{}, fmt.Errorf("read MAVLink UDP frame: %w", err)
		}
		data := make([]byte, n)
		copy(data, buf[:n])

		frames := s.framesFromDatagram(data)
		if len(frames) == 0 {
			continue
		}
		s.pending = append(s.pending, frames[1:]...)
		return frames[0], nil
	}
}

func (s *UDPSource) framesFromDatagram(data []byte) []RawFrame {
	var frames []RawFrame
	for offset := 0; offset < len(data); {
		frameLen, ok := mavlinkFrameLen(data[offset:])
		if !ok {
			offset++
			continue
		}
		if len(data[offset:]) < frameLen {
			break
		}

		frameBytes := make([]byte, frameLen)
		copy(frameBytes, data[offset:offset+frameLen])
		if frame, err := DecodeFrame(frameBytes); err == nil {
			vehicleID := fmt.Sprintf("sys-%03d", frame.SystemID)
			frames = append(frames, RawFrame{
				Subject:   fmt.Sprintf("%s.%s", s.subjectPrefix, vehicleID),
				VehicleID: vehicleID,
				SystemID:  frame.SystemID,
				EmittedAt: s.clock(),
				Bytes:     frameBytes,
			})
		}
		offset += frameLen
	}
	return frames
}

func mavlinkFrameLen(data []byte) (int, bool) {
	if len(data) < headerLenV1+checksumLen {
		return 0, false
	}
	switch data[0] {
	case stxV1:
		return headerLenV1 + int(data[1]) + checksumLen, true
	case stxV2:
		if len(data) < headerLenV2+checksumLen {
			return 0, false
		}
		frameLen := headerLenV2 + int(data[1]) + checksumLen
		if data[2]&incompatFlagSigned != 0 {
			frameLen += signatureLen
		}
		return frameLen, true
	default:
		return 0, false
	}
}

func isTimeout(err error) bool {
	var netErr net.Error
	return errors.As(err, &netErr) && netErr.Timeout()
}

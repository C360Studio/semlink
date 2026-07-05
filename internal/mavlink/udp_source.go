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
	conn, err := net.ListenUDP("udp", udpAddr)
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
		maxDatagramBytes = 280
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

		frame, err := DecodeFrame(data)
		if err != nil {
			continue
		}
		vehicleID := fmt.Sprintf("sys-%03d", frame.SystemID)
		return RawFrame{
			Subject:   fmt.Sprintf("%s.%s", s.subjectPrefix, vehicleID),
			VehicleID: vehicleID,
			SystemID:  frame.SystemID,
			EmittedAt: s.clock(),
			Bytes:     data,
		}, nil
	}
}

func isTimeout(err error) bool {
	var netErr net.Error
	return errors.As(err, &netErr) && netErr.Timeout()
}

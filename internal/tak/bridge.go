package tak

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"sync"
	"time"

	"github.com/c360studio/semlink/internal/cop"
	"github.com/c360studio/semlink/internal/cot"
	"github.com/c360studio/semlink/internal/gcs"
	"github.com/c360studio/semlink/internal/graphprojection"
	semruntime "github.com/c360studio/semlink/internal/semstreams"
)

const (
	DefaultMulticastAddr = "239.2.3.1:6969"
	DefaultInterval      = time.Second
	defaultWriteTimeout  = time.Second
	defaultGraphTimeout  = 2 * time.Second
)

type Store interface {
	Snapshot() gcs.Snapshot
	ApplyCOPView(cop.View)
	RecordProjectedWrite()
	RecordGraphWrite(entityID string, revision uint64, latency time.Duration)
	RecordGraphError()
}

type GraphWriter interface {
	UpsertProjection(ctx context.Context, p graphprojection.Projection) (*semruntime.WriteResult, error)
}

type Config struct {
	MulticastAddr  string
	TCPListenAddr  string
	InboundUDPAddr string
	InboundTCPAddr string
	Interval       time.Duration
	Logger         *slog.Logger
}

type Bridge struct {
	cfg        Config
	store      Store
	graph      GraphWriter
	translator *cop.Translator
	logger     *slog.Logger

	mu      sync.Mutex
	clients map[net.Conn]struct{}
}

func NewBridge(cfg Config, store Store, graph GraphWriter) (*Bridge, error) {
	if store == nil {
		return nil, errors.New("tak bridge: nil store")
	}
	if cfg.Interval <= 0 {
		cfg.Interval = DefaultInterval
	}
	logger := cfg.Logger
	if logger == nil {
		logger = slog.Default()
	}
	return &Bridge{
		cfg:        cfg,
		store:      store,
		graph:      graph,
		translator: cop.NewTranslator(),
		logger:     logger,
		clients:    make(map[net.Conn]struct{}),
	}, nil
}

func (b *Bridge) Start(ctx context.Context) {
	if b.cfg.MulticastAddr != "" || b.cfg.TCPListenAddr != "" {
		go b.runOutbound(ctx)
	}
	if b.cfg.TCPListenAddr != "" {
		go b.runTCPServer(ctx, b.cfg.TCPListenAddr, b.acceptOutboundClient)
	}
	if b.cfg.InboundUDPAddr != "" {
		go b.runInboundUDP(ctx, b.cfg.InboundUDPAddr)
	}
	if b.cfg.InboundTCPAddr != "" {
		go b.runTCPServer(ctx, b.cfg.InboundTCPAddr, b.handleInboundTCPConn)
	}
}

func EventsFromSnapshot(snapshot gcs.Snapshot, now time.Time) []cot.Event {
	vehiclesByID := make(map[string]gcs.VehicleView, len(snapshot.Vehicles))
	events := make([]cot.Event, 0, len(snapshot.Vehicles)+len(snapshot.Alerts))
	for _, vehicle := range snapshot.Vehicles {
		vehiclesByID[vehicle.EntityID] = vehicle
		if vehicle.LastSeen.IsZero() {
			continue
		}
		events = append(events, cot.Event{
			UID:       vehicleToken(vehicle),
			Type:      cot.TypeAirTrack,
			Time:      now,
			Point:     &cot.Point{Lat: vehicle.LatitudeDeg, Lon: vehicle.LongitudeDeg, HAE: vehicle.AltitudeM, CE: 10, LE: 20},
			Callsign:  vehicle.Callsign,
			CourseDeg: vehicle.HeadingDeg,
			SpeedMPS:  vehicle.GroundSpeedMS,
			HasTrack:  true,
		})
	}
	for _, alert := range snapshot.Alerts {
		if !alert.Active {
			continue
		}
		vehicle, ok := vehiclesByID[alert.SubjectEntity]
		if !ok {
			continue
		}
		events = append(events, cot.Event{
			UID:      alertUID(alert),
			Type:     cot.TypeAlert,
			Time:     now,
			Point:    &cot.Point{Lat: vehicle.LatitudeDeg, Lon: vehicle.LongitudeDeg, HAE: vehicle.AltitudeM, CE: 10, LE: 20},
			Callsign: fmt.Sprintf("%s %s", vehicle.Callsign, alert.Kind),
			Remarks:  alert.Message,
		})
	}
	return events
}

func (b *Bridge) runOutbound(ctx context.Context) {
	var udpConn net.Conn
	if b.cfg.MulticastAddr != "" {
		conn, err := net.Dial("udp", b.cfg.MulticastAddr)
		if err != nil {
			b.logger.Warn("tak multicast disabled", slog.String("addr", b.cfg.MulticastAddr), slog.Any("error", err))
		} else {
			udpConn = conn
			defer udpConn.Close()
		}
	}
	ticker := time.NewTicker(b.cfg.Interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			b.closeClients()
			return
		case now := <-ticker.C:
			for _, event := range EventsFromSnapshot(b.store.Snapshot(), now) {
				raw, err := cot.Marshal(event)
				if err != nil {
					b.logger.Debug("cot marshal failed", slog.String("uid", event.UID), slog.Any("error", err))
					continue
				}
				if udpConn != nil {
					_ = udpConn.SetWriteDeadline(time.Now().Add(defaultWriteTimeout))
					if _, err := udpConn.Write(raw); err != nil {
						b.logger.Debug("tak multicast write failed", slog.Any("error", err))
					}
				}
				b.writeTCP(raw)
			}
		}
	}
}

func (b *Bridge) runTCPServer(ctx context.Context, addr string, handler func(context.Context, net.Conn)) {
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		b.logger.Warn("tak tcp listener disabled", slog.String("addr", addr), slog.Any("error", err))
		return
	}
	defer listener.Close()
	go func() {
		<-ctx.Done()
		_ = listener.Close()
	}()
	for {
		conn, err := listener.Accept()
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			b.logger.Debug("tak tcp accept failed", slog.Any("error", err))
			continue
		}
		go handler(ctx, conn)
	}
}

func (b *Bridge) acceptOutboundClient(ctx context.Context, conn net.Conn) {
	b.mu.Lock()
	b.clients[conn] = struct{}{}
	b.mu.Unlock()
	go func() {
		<-ctx.Done()
		_ = conn.Close()
	}()
}

func (b *Bridge) writeTCP(raw []byte) {
	b.mu.Lock()
	defer b.mu.Unlock()
	for conn := range b.clients {
		_ = conn.SetWriteDeadline(time.Now().Add(defaultWriteTimeout))
		if _, err := conn.Write(append(append([]byte(nil), raw...), '\n')); err != nil {
			_ = conn.Close()
			delete(b.clients, conn)
		}
	}
}

func (b *Bridge) closeClients() {
	b.mu.Lock()
	defer b.mu.Unlock()
	for conn := range b.clients {
		_ = conn.Close()
		delete(b.clients, conn)
	}
}

func (b *Bridge) runInboundUDP(ctx context.Context, addr string) {
	udpAddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		b.logger.Warn("tak inbound udp disabled", slog.String("addr", addr), slog.Any("error", err))
		return
	}
	conn, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		b.logger.Warn("tak inbound udp disabled", slog.String("addr", addr), slog.Any("error", err))
		return
	}
	defer conn.Close()
	go func() {
		<-ctx.Done()
		_ = conn.Close()
	}()
	buf := make([]byte, 64*1024)
	for {
		n, _, err := conn.ReadFromUDP(buf)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			b.logger.Debug("tak inbound udp read failed", slog.Any("error", err))
			continue
		}
		b.handleInbound(ctx, append([]byte(nil), buf[:n]...))
	}
}

func (b *Bridge) handleInboundTCPConn(ctx context.Context, conn net.Conn) {
	defer conn.Close()
	scanner := bufio.NewScanner(conn)
	scanner.Buffer(make([]byte, 1024), 1024*1024)
	for scanner.Scan() {
		b.handleInbound(ctx, append([]byte(nil), scanner.Bytes()...))
	}
	if err := scanner.Err(); err != nil && ctx.Err() == nil {
		b.logger.Debug("tak inbound tcp scan failed", slog.Any("error", err))
	}
}

func (b *Bridge) handleInbound(ctx context.Context, raw []byte) {
	if b.graph == nil {
		b.logger.Debug("tak inbound dropped because graph writer is disabled")
		return
	}
	event, err := cot.Unmarshal(raw)
	if err != nil {
		b.logger.Debug("tak inbound decode failed", slog.Any("error", err))
		return
	}
	result, ok, err := b.translator.Apply(event, time.Now())
	if err != nil {
		b.logger.Debug("tak inbound projection failed", slog.String("uid", event.UID), slog.Any("error", err))
		return
	}
	if !ok {
		return
	}
	b.store.ApplyCOPView(result.View)
	b.store.RecordProjectedWrite()
	writeCtx, cancel := context.WithTimeout(ctx, defaultGraphTimeout)
	start := time.Now()
	write, err := b.graph.UpsertProjection(writeCtx, result.Projection)
	cancel()
	if err != nil {
		b.store.RecordGraphError()
		b.logger.Debug("tak inbound graph write failed", slog.String("entity", result.Projection.Entity.ID), slog.Any("error", err))
		return
	}
	b.store.RecordGraphWrite(result.Projection.Entity.ID, write.Revision, time.Since(start))
}

func vehicleToken(vehicle gcs.VehicleView) string {
	if vehicle.SystemID > 0 {
		return fmt.Sprintf("uav-%03d", vehicle.SystemID)
	}
	return vehicle.Callsign
}

func alertUID(alert gcs.AlertView) string {
	return "semlink-alert-" + lastToken(alert.EntityID)
}

func lastToken(id string) string {
	if i := len(id) - 1; i >= 0 {
		for j := len(id) - 1; j >= 0; j-- {
			if id[j] == '.' {
				return id[j+1:]
			}
		}
	}
	return id
}

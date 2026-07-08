package gcs

import (
	"context"
	"log/slog"
	"strings"
	"time"

	"github.com/c360studio/semlink/internal/mavlink"
	"github.com/c360studio/semlink/internal/projector"
	semruntime "github.com/c360studio/semlink/internal/semstreams"
	"github.com/c360studio/semstreams/pkg/buffer"
)

type DemoConfig struct {
	Vehicles         int
	Hz               int
	BufferCapacity   int
	MAVLinkUDPListen string
	Logger           *slog.Logger
}

type TelemetrySourceMode string

const (
	TelemetrySourceInternalSimulator  TelemetrySourceMode = "internal-simulator"
	TelemetrySourceExternalMAVLinkUDP TelemetrySourceMode = "external-mavlink-udp"
)

type Demo struct {
	cfg       DemoConfig
	runtime   *semruntime.Runtime
	projector *projector.Projector
	store     *Store
	buffer    buffer.Buffer[mavlink.RawFrame]
	logger    *slog.Logger
}

func (cfg DemoConfig) TelemetrySourceMode() TelemetrySourceMode {
	if strings.TrimSpace(cfg.MAVLinkUDPListen) != "" {
		return TelemetrySourceExternalMAVLinkUDP
	}
	return TelemetrySourceInternalSimulator
}

func NewDemo(cfg DemoConfig, rt *semruntime.Runtime, store *Store) (*Demo, error) {
	if cfg.Vehicles <= 0 {
		cfg.Vehicles = 12
	}
	if cfg.Hz <= 0 {
		cfg.Hz = 20
	}
	if cfg.BufferCapacity <= 0 {
		cfg.BufferCapacity = 10_000
	}
	logger := cfg.Logger
	if logger == nil {
		logger = slog.Default()
	}
	rawBuffer, err := buffer.NewCircularBuffer[mavlink.RawFrame](cfg.BufferCapacity, buffer.WithOverflowPolicy[mavlink.RawFrame](buffer.DropOldest))
	if err != nil {
		return nil, err
	}
	return &Demo{
		cfg:       cfg,
		runtime:   rt,
		projector: projector.New(projector.DefaultConfig()),
		store:     store,
		buffer:    rawBuffer,
		logger:    logger,
	}, nil
}

func (d *Demo) Start(ctx context.Context) {
	if d.cfg.TelemetrySourceMode() == TelemetrySourceExternalMAVLinkUDP {
		go d.runUDPSource(ctx)
	} else {
		go d.runSimulator(ctx)
	}
	go d.runProjector(ctx)
	go d.runLinkMonitor(ctx)
}

func (d *Demo) runUDPSource(ctx context.Context) {
	source, err := mavlink.ListenUDP(mavlink.UDPSourceConfig{
		ListenAddr:    d.cfg.MAVLinkUDPListen,
		SubjectPrefix: "mavlink.raw.ardupilot-sitl",
	})
	if err != nil {
		d.logger.Error("mavlink udp source failed", slog.String("listen", d.cfg.MAVLinkUDPListen), slog.Any("error", err))
		return
	}
	defer source.Close()
	d.logger.Info("mavlink udp source enabled", slog.String("listen", source.LocalAddr().String()))

	for {
		frame, err := source.Read(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			d.logger.Warn("mavlink udp read failed", slog.Any("error", err))
			continue
		}
		d.recordRawFrame(frame)
	}
}

func (d *Demo) runSimulator(ctx context.Context) {
	sim := mavlink.NewSimulator(d.cfg.Vehicles)
	interval := time.Second / time.Duration(d.cfg.Hz)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case now := <-ticker.C:
			frames, err := sim.Next(now)
			if err != nil {
				d.logger.Warn("simulator failed", slog.Any("error", err))
				continue
			}
			for _, frame := range frames {
				d.recordRawFrame(frame)
			}
			stats := d.buffer.Stats()
			d.store.RecordBuffer(d.buffer.Size(), d.buffer.Capacity(), stats.Drops())
		}
	}
}

func (d *Demo) recordRawFrame(frame mavlink.RawFrame) {
	if err := d.buffer.Write(frame); err != nil {
		d.logger.Warn("raw buffer write failed", slog.Any("error", err))
		return
	}
	d.store.RecordRawFrame()
	stats := d.buffer.Stats()
	d.store.RecordBuffer(d.buffer.Size(), d.buffer.Capacity(), stats.Drops())
}

func (d *Demo) runProjector(ctx context.Context) {
	idle := time.NewTicker(5 * time.Millisecond)
	defer idle.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-idle.C:
			batch := d.buffer.ReadBatch(256)
			if len(batch) == 0 {
				continue
			}
			for _, frame := range batch {
				d.handleFrame(ctx, frame)
			}
			vehicles, alerts := d.projector.Snapshot()
			d.store.ApplyProjectorSnapshot(vehicles, alerts)
			stats := d.buffer.Stats()
			d.store.RecordBuffer(d.buffer.Size(), d.buffer.Capacity(), stats.Drops())
		}
	}
}

func (d *Demo) handleFrame(ctx context.Context, frame mavlink.RawFrame) {
	if err := semruntime.PublishRawFrame(ctx, d.runtime.Client, frame.Subject, frame.Bytes); err != nil {
		d.store.RecordRawPublishError()
		d.logger.Debug("raw frame publish failed", slog.Any("error", err))
	}
	msg, err := mavlink.DecodeMessage(frame.Bytes)
	if err != nil {
		d.store.RecordDecodeError()
		d.logger.Debug("mavlink decode failed", slog.Any("error", err))
		return
	}
	d.store.RecordDecodedFrame()
	projections := d.projector.Apply(msg, frame.EmittedAt)
	d.writeProjections(ctx, projections)
}

func (d *Demo) runLinkMonitor(ctx context.Context) {
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case now := <-ticker.C:
			projections := d.projector.CheckLinkTimeouts(now)
			d.writeProjections(ctx, projections)
			vehicles, alerts := d.projector.Snapshot()
			d.store.ApplyProjectorSnapshot(vehicles, alerts)
		}
	}
}

func (d *Demo) writeProjections(ctx context.Context, projections []projector.Projection) {
	for _, proj := range projections {
		d.store.RecordProjectedWrite()
		writeCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
		start := time.Now()
		result, err := d.runtime.Graph.UpsertProjection(writeCtx, proj)
		cancel()
		if err != nil {
			d.store.RecordGraphError()
			d.logger.Debug("semstreams projection write failed", slog.String("entity", proj.Entity.ID), slog.Any("error", err))
			continue
		}
		d.store.RecordGraphWrite(proj.Entity.ID, result.Revision, time.Since(start))
	}
}

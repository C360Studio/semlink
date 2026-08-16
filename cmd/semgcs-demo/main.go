package main

import (
	"context"
	"flag"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"syscall"
	"time"

	csbridge "github.com/c360studio/semlink/internal/csapi"
	"github.com/c360studio/semlink/internal/gcs"
	"github.com/c360studio/semlink/internal/handoff"
	semruntime "github.com/c360studio/semlink/internal/semstreams"
	"github.com/c360studio/semlink/internal/tak"
)

func main() {
	var (
		listen         = flag.String("listen", ":8080", "HTTP listen address")
		natsURL        = flag.String("nats-url", getenv("NATS_URL", "nats://127.0.0.1:4222"), "NATS URL used when -embedded-nats=false")
		embeddedNATS   = flag.Bool("embedded-nats", true, "start a local NATS JetStream container for the demo")
		stateDir       = flag.String("state-dir", getenv("SEMLINK_NATS_STATE_DIR", ""), "persistent embedded NATS state directory; empty uses an owned development temporary directory")
		vehicles       = flag.Int("vehicles", 12, "number of simulated vehicles")
		hz             = flag.Int("hz", 20, "simulator ticks per second")
		bufferCapacity = flag.Int("buffer", 10000, "raw telemetry buffer capacity")
		mavlinkUDP     = flag.String("mavlink-udp", getenv("MAVLINK_UDP_LISTEN", ""), "optional external MAVLink UDP listen address; when set, disables the internal simulator")
		staticDir      = flag.String("static", filepath.Join("ui", "dist"), "built UI static directory")
		csapiURL       = flag.String("csapi-url", getenv("CS_API_URL", ""), "optional SemConnect CS API base URL for standards projection")
		csapiInterval  = flag.Duration("csapi-interval", 2*time.Second, "SemConnect CS API bridge sync interval")
		csapiObsEvery  = flag.Duration("csapi-observation-interval", 5*time.Second, "minimum interval between CS API observations per datastream")
		takEnabled     = flag.Bool("tak", getenvBool("TAK_ENABLED", false), "enable TAK/CoT outbound multicast bridge")
		takMulticast   = flag.String("tak-multicast", getenv("TAK_MULTICAST_ADDR", tak.DefaultMulticastAddr), "TAK outbound UDP multicast address")
		takTCP         = flag.String("tak-tcp", getenv("TAK_TCP_LISTEN", ""), "optional TAK outbound TCP listen address")
		takInboundUDP  = flag.String("tak-inbound-udp", getenv("TAK_INBOUND_UDP", ""), "optional TAK inbound UDP listen address")
		takInboundTCP  = flag.String("tak-inbound-tcp", getenv("TAK_INBOUND_TCP", ""), "optional TAK inbound TCP listen address")
		takInterval    = flag.Duration("tak-interval", tak.DefaultInterval, "TAK outbound publish interval")
	)
	flag.Parse()

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	profile, err := handoff.ParseEnv(handoffProfileValues(os.Environ(), runtimeProfileValues{
		Listen:                *listen,
		EmbeddedNATS:          *embeddedNATS,
		NATSURL:               *natsURL,
		Vehicles:              *vehicles,
		Hz:                    *hz,
		Buffer:                *bufferCapacity,
		MAVLinkUDP:            *mavlinkUDP,
		CSAPIURL:              *csapiURL,
		CSAPIInterval:         *csapiInterval,
		CSAPIObservationEvery: *csapiObsEvery,
		TAKEnabled:            *takEnabled,
		TAKMulticast:          *takMulticast,
		TAKTCP:                *takTCP,
		TAKInboundUDP:         *takInboundUDP,
		TAKInboundTCP:         *takInboundTCP,
		TAKInterval:           *takInterval,
	}))
	if err != nil {
		logger.Error("invalid companion handoff profile", slog.Any("error", err))
		os.Exit(1)
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	rt, err := semruntime.StartRuntime(ctx, semruntime.RuntimeOptions{
		NATSURL:  *natsURL,
		Embedded: *embeddedNATS,
		StateDir: *stateDir,
		Logger:   logger,
	})
	if err != nil {
		logger.Error("failed to start SemStreams runtime", slog.Any("error", err))
		os.Exit(1)
	}
	defer func() {
		stopCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := rt.Stop(stopCtx); err != nil {
			logger.Warn("runtime stop failed", slog.Any("error", err))
		}
	}()

	store := gcs.NewStore(rt.NATSURL, *embeddedNATS)
	store.SetSemStreamsStatePosture(rt.StateSchemaVersion, rt.FreshState, rt.ReusedState)
	if *csapiURL != "" {
		bridge, err := csbridge.NewBridge(csbridge.Config{
			BaseURL:             *csapiURL,
			Interval:            *csapiInterval,
			ObservationInterval: *csapiObsEvery,
			Logger:              logger,
		}, store)
		if err != nil {
			logger.Error("failed to create CS API bridge", slog.Any("error", err))
			os.Exit(1)
		}
		bridge.Start(ctx)
		logger.Info("SemConnect CS API bridge enabled",
			slog.String("csapi_url", *csapiURL),
			slog.Duration("sync_interval", *csapiInterval),
			slog.Duration("observation_interval", *csapiObsEvery))
	}
	if *takEnabled || *takTCP != "" || *takInboundUDP != "" || *takInboundTCP != "" {
		multicast := ""
		if *takEnabled {
			multicast = *takMulticast
		}
		bridge, err := tak.NewBridge(tak.Config{
			MulticastAddr:  multicast,
			TCPListenAddr:  *takTCP,
			InboundUDPAddr: *takInboundUDP,
			InboundTCPAddr: *takInboundTCP,
			Interval:       *takInterval,
			Logger:         logger,
		}, store, rt.Graph)
		if err != nil {
			logger.Error("failed to create TAK bridge", slog.Any("error", err))
			os.Exit(1)
		}
		bridge.Start(ctx)
		logger.Info("TAK/CoT bridge enabled",
			slog.String("multicast", multicast),
			slog.String("tcp", *takTCP),
			slog.String("inbound_udp", *takInboundUDP),
			slog.String("inbound_tcp", *takInboundTCP),
			slog.Duration("interval", *takInterval))
	}
	demo, err := gcs.NewDemo(gcs.DemoConfig{
		Vehicles:         *vehicles,
		Hz:               *hz,
		BufferCapacity:   *bufferCapacity,
		MAVLinkUDPListen: *mavlinkUDP,
		Logger:           logger,
	}, rt, store)
	if err != nil {
		logger.Error("failed to create demo", slog.Any("error", err))
		os.Exit(1)
	}
	demo.Start(ctx)

	commands := gcs.NewCommandService(rt.Graph, store)
	server := &http.Server{
		Addr: *listen,
		Handler: gcs.NewServer(store, commands, *staticDir, gcs.ServerOptions{
			Graph:          rt.Graph,
			CSAPIURL:       *csapiURL,
			NodeID:         profile.NodeID,
			HandoffProfile: &profile,
		}).Handler(),
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		_ = server.Shutdown(shutdownCtx)
	}()

	logger.Info("semgcs demo listening",
		slog.String("listen", *listen),
		slog.String("nats_url", rt.NATSURL),
		slog.Bool("embedded_nats", *embeddedNATS),
		slog.Int("vehicles", *vehicles),
		slog.Int("hz", *hz),
		slog.String("mavlink_udp", *mavlinkUDP))

	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logger.Error("http server failed", slog.Any("error", err))
		os.Exit(1)
	}
}

type runtimeProfileValues struct {
	Listen                string
	EmbeddedNATS          bool
	NATSURL               string
	Vehicles              int
	Hz                    int
	Buffer                int
	MAVLinkUDP            string
	CSAPIURL              string
	CSAPIInterval         time.Duration
	CSAPIObservationEvery time.Duration
	TAKEnabled            bool
	TAKMulticast          string
	TAKTCP                string
	TAKInboundUDP         string
	TAKInboundTCP         string
	TAKInterval           time.Duration
}

func handoffProfileValues(environ []string, runtime runtimeProfileValues) map[string]string {
	values := handoff.MapFromEnviron(environ)
	values[handoff.EnvHTTPListen] = runtime.Listen
	values[handoff.EnvEmbeddedNATS] = strconv.FormatBool(runtime.EmbeddedNATS)
	values[handoff.EnvNATSURL] = runtime.NATSURL
	values[handoff.EnvVehicles] = strconv.Itoa(runtime.Vehicles)
	values[handoff.EnvHz] = strconv.Itoa(runtime.Hz)
	values[handoff.EnvBuffer] = strconv.Itoa(runtime.Buffer)
	values[handoff.EnvMAVLinkUDPListen] = runtime.MAVLinkUDP
	values[handoff.EnvCSAPIURL] = runtime.CSAPIURL
	values[handoff.EnvCSAPIInterval] = runtime.CSAPIInterval.String()
	values[handoff.EnvCSAPIObservation] = runtime.CSAPIObservationEvery.String()
	values[handoff.EnvTAKEnabled] = strconv.FormatBool(runtime.TAKEnabled)
	values[handoff.EnvTAKMulticastAddr] = runtime.TAKMulticast
	values[handoff.EnvTAKTCPListen] = runtime.TAKTCP
	values[handoff.EnvTAKInboundUDPListen] = runtime.TAKInboundUDP
	values[handoff.EnvTAKInboundTCPListen] = runtime.TAKInboundTCP
	values[handoff.EnvTAKInterval] = runtime.TAKInterval.String()
	return values
}

func getenv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func getenvBool(key string, fallback bool) bool {
	switch os.Getenv(key) {
	case "1", "true", "TRUE", "yes", "YES", "on", "ON":
		return true
	case "0", "false", "FALSE", "no", "NO", "off", "OFF":
		return false
	default:
		return fallback
	}
}

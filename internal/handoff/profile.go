package handoff

import (
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"
	"time"
	"unicode"
)

const (
	EnvNodeID                  = "SEMLINK_NODE_ID"
	EnvVehicleID               = "SEMLINK_VEHICLE_ID"
	EnvCallsign                = "SEMLINK_CALLSIGN"
	EnvHTTPListen              = "SEMLINK_HTTP_LISTEN"
	EnvBlueOSHostPort          = "SEMLINK_BLUEOS_HOST_PORT"
	EnvEmbeddedNATS            = "SEMLINK_EMBEDDED_NATS"
	EnvNATSURL                 = "NATS_URL"
	EnvMAVLinkUDPListen        = "SEMLINK_MAVLINK_UDP_LISTEN"
	EnvMAVLinkUDPHost          = "SEMLINK_MAVLINK_UDP_HOST"
	EnvMAVLinkUDPPort          = "SEMLINK_MAVLINK_UDP_PORT"
	EnvVehicles                = "SEMLINK_VEHICLES"
	EnvHz                      = "SEMLINK_HZ"
	EnvBuffer                  = "SEMLINK_BUFFER"
	EnvMeshPeers               = "SEMLINK_MESH_PEERS"
	EnvCSAPIURL                = "CS_API_URL"
	EnvCSAPIInterval           = "SEMLINK_CSAPI_INTERVAL"
	EnvCSAPIObservation        = "SEMLINK_CSAPI_OBSERVATION_INTERVAL"
	EnvCommandRuntimeMode      = "SEMLINK_COMMAND_RUNTIME_MODE"
	EnvHardwareTransmitEnabled = "SEMLINK_HARDWARE_TRANSMIT_ENABLED"
	EnvTAKEnabled              = "SEMLINK_TAK_ENABLED"
	EnvTAKMulticastAddr        = "SEMLINK_TAK_MULTICAST_ADDR"
	EnvTAKTCPListen            = "SEMLINK_TAK_TCP_LISTEN"
	EnvTAKInboundUDPListen     = "SEMLINK_TAK_INBOUND_UDP_LISTEN"
	EnvTAKInboundTCPListen     = "SEMLINK_TAK_INBOUND_TCP_LISTEN"
	EnvTAKInterval             = "SEMLINK_TAK_INTERVAL"
)

type CommandRuntimeMode string

const (
	CommandRuntimeHardwareReadonly CommandRuntimeMode = "hardware-readonly"
	CommandRuntimeSimulator        CommandRuntimeMode = "simulator"
)

type Profile struct {
	NodeID                   string
	VehicleID                string
	Callsign                 string
	HTTPListen               string
	BlueOSHostPort           int
	EmbeddedNATS             bool
	NATSURL                  string
	MAVLinkUDPListen         string
	MAVLinkUDPHost           string
	MAVLinkUDPPort           int
	Vehicles                 int
	Hz                       int
	Buffer                   int
	MeshPeers                []string
	CSAPIURL                 string
	CSAPIInterval            time.Duration
	CSAPIObservationInterval time.Duration
	CommandRuntimeMode       CommandRuntimeMode
	HardwareTransmitEnabled  bool
	TAKEnabled               bool
	TAKMulticastAddr         string
	TAKTCPListen             string
	TAKInboundUDPListen      string
	TAKInboundTCPListen      string
	TAKInterval              time.Duration
}

type Problem struct {
	Field   string
	Message string
}

type ValidationError struct {
	Problems []Problem
}

func (e *ValidationError) Error() string {
	if e == nil || len(e.Problems) == 0 {
		return "invalid companion handoff profile"
	}
	parts := make([]string, 0, len(e.Problems))
	for _, problem := range e.Problems {
		parts = append(parts, fmt.Sprintf("%s: %s", problem.Field, problem.Message))
	}
	return "invalid companion handoff profile: " + strings.Join(parts, "; ")
}

func DefaultProfile() Profile {
	return Profile{
		NodeID:                   "semlink-local",
		VehicleID:                "semlink-local",
		Callsign:                 "SEMLINK-LOCAL",
		HTTPListen:               ":80",
		BlueOSHostPort:           8081,
		EmbeddedNATS:             true,
		NATSURL:                  "nats://127.0.0.1:4222",
		MAVLinkUDPListen:         ":14550",
		MAVLinkUDPHost:           "127.0.0.1",
		MAVLinkUDPPort:           14550,
		Vehicles:                 1,
		Hz:                       5,
		Buffer:                   10_000,
		CSAPIInterval:            2 * time.Second,
		CSAPIObservationInterval: 5 * time.Second,
		CommandRuntimeMode:       CommandRuntimeHardwareReadonly,
		TAKMulticastAddr:         "239.2.3.1:6969",
		TAKInterval:              time.Second,
	}
}

func ParseEnv(values map[string]string) (Profile, error) {
	profile := DefaultProfile()
	var problems []Problem

	profile.NodeID = readRequiredToken(values, EnvNodeID, profile.NodeID, &problems)
	profile.VehicleID = readRequiredToken(values, EnvVehicleID, profile.NodeID, &problems)
	profile.Callsign = readRequiredString(values, EnvCallsign, strings.ToUpper(profile.NodeID), &problems)
	profile.HTTPListen = readListenAddr(values, EnvHTTPListen, profile.HTTPListen, true, &problems)
	profile.BlueOSHostPort = readInt(values, EnvBlueOSHostPort, profile.BlueOSHostPort, 1, 65535, &problems)
	profile.EmbeddedNATS = readBool(values, EnvEmbeddedNATS, profile.EmbeddedNATS, &problems)
	profile.NATSURL = readNATSURL(values, profile.NATSURL, profile.EmbeddedNATS, &problems)
	profile.MAVLinkUDPListen = readListenAddr(values, EnvMAVLinkUDPListen, profile.MAVLinkUDPListen, false, &problems)
	profile.MAVLinkUDPHost = readRequiredString(values, EnvMAVLinkUDPHost, profile.MAVLinkUDPHost, &problems)
	if hasWhitespace(profile.MAVLinkUDPHost) {
		problems = append(problems, Problem{Field: EnvMAVLinkUDPHost, Message: "must be a hostname or IP address without whitespace"})
	}
	profile.MAVLinkUDPPort = readInt(values, EnvMAVLinkUDPPort, profile.MAVLinkUDPPort, 1, 65535, &problems)
	profile.Vehicles = readInt(values, EnvVehicles, profile.Vehicles, 1, 255, &problems)
	profile.Hz = readInt(values, EnvHz, profile.Hz, 1, 1000, &problems)
	profile.Buffer = readInt(values, EnvBuffer, profile.Buffer, 1, 10_000_000, &problems)
	profile.MeshPeers = readURLList(values, EnvMeshPeers, []string{"http", "https"}, &problems)
	profile.CSAPIURL = readURL(values, EnvCSAPIURL, "", []string{"http", "https"}, true, &problems)
	profile.CSAPIInterval = readDuration(values, EnvCSAPIInterval, profile.CSAPIInterval, &problems)
	profile.CSAPIObservationInterval = readDuration(values, EnvCSAPIObservation, profile.CSAPIObservationInterval, &problems)
	profile.CommandRuntimeMode = readCommandRuntimeMode(values, EnvCommandRuntimeMode, profile.CommandRuntimeMode, &problems)
	profile.HardwareTransmitEnabled = readBool(values, EnvHardwareTransmitEnabled, profile.HardwareTransmitEnabled, &problems)
	if profile.HardwareTransmitEnabled {
		problems = append(problems, Problem{
			Field:   EnvHardwareTransmitEnabled,
			Message: "must be false until a hardware command OpenSpec change is accepted",
		})
	}
	profile.TAKEnabled = readBool(values, EnvTAKEnabled, profile.TAKEnabled, &problems)
	profile.TAKMulticastAddr = readListenAddr(values, EnvTAKMulticastAddr, profile.TAKMulticastAddr, false, &problems)
	profile.TAKTCPListen = readListenAddr(values, EnvTAKTCPListen, profile.TAKTCPListen, false, &problems)
	profile.TAKInboundUDPListen = readListenAddr(values, EnvTAKInboundUDPListen, profile.TAKInboundUDPListen, false, &problems)
	profile.TAKInboundTCPListen = readListenAddr(values, EnvTAKInboundTCPListen, profile.TAKInboundTCPListen, false, &problems)
	profile.TAKInterval = readDuration(values, EnvTAKInterval, profile.TAKInterval, &problems)

	if len(problems) > 0 {
		return Profile{}, &ValidationError{Problems: problems}
	}
	return profile, nil
}

func readRequiredString(values map[string]string, field, fallback string, problems *[]Problem) string {
	raw, ok := values[field]
	if !ok {
		return fallback
	}
	value := strings.TrimSpace(raw)
	if value == "" {
		*problems = append(*problems, Problem{Field: field, Message: "must not be empty"})
		return fallback
	}
	return value
}

func readRequiredToken(values map[string]string, field, fallback string, problems *[]Problem) string {
	value := readRequiredString(values, field, fallback, problems)
	if hasWhitespace(value) || hasControl(value) {
		*problems = append(*problems, Problem{Field: field, Message: "must be a stable token without whitespace or control characters"})
	}
	return value
}

func readBool(values map[string]string, field string, fallback bool, problems *[]Problem) bool {
	raw, ok := values[field]
	if !ok {
		return fallback
	}
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "1", "t", "true", "yes", "on":
		return true
	case "0", "f", "false", "no", "off":
		return false
	default:
		*problems = append(*problems, Problem{Field: field, Message: "must be a boolean value: true or false"})
		return fallback
	}
}

func readNATSURL(values map[string]string, fallback string, embeddedNATS bool, problems *[]Problem) string {
	raw, ok := values[EnvNATSURL]
	if !ok {
		return fallback
	}
	value := strings.TrimSpace(raw)
	if value == "" {
		message := "must not be empty"
		if !embeddedNATS {
			message = "must be set when SEMLINK_EMBEDDED_NATS is false"
		}
		*problems = append(*problems, Problem{Field: EnvNATSURL, Message: message})
		return ""
	}
	if err := validateURL(value, []string{"nats", "tls", "ws", "wss"}); err != nil {
		*problems = append(*problems, Problem{Field: EnvNATSURL, Message: err.Error()})
		return fallback
	}
	return value
}

func readInt(values map[string]string, field string, fallback, minValue, maxValue int, problems *[]Problem) int {
	raw, ok := values[field]
	if !ok {
		return fallback
	}
	value, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil {
		*problems = append(*problems, Problem{Field: field, Message: fmt.Sprintf("must be an integer from %d to %d", minValue, maxValue)})
		return fallback
	}
	if value < minValue || value > maxValue {
		*problems = append(*problems, Problem{Field: field, Message: fmt.Sprintf("must be from %d to %d", minValue, maxValue)})
		return fallback
	}
	return value
}

func readDuration(values map[string]string, field string, fallback time.Duration, problems *[]Problem) time.Duration {
	raw, ok := values[field]
	if !ok {
		return fallback
	}
	value, err := time.ParseDuration(strings.TrimSpace(raw))
	if err != nil || value <= 0 {
		*problems = append(*problems, Problem{Field: field, Message: "must be a positive duration such as 2s or 500ms"})
		return fallback
	}
	return value
}

func readListenAddr(values map[string]string, field, fallback string, required bool, problems *[]Problem) string {
	raw, ok := values[field]
	if !ok {
		return fallback
	}
	value := strings.TrimSpace(raw)
	if value == "" {
		if required {
			*problems = append(*problems, Problem{Field: field, Message: "must not be empty"})
			return fallback
		}
		return ""
	}
	if err := validateHostPort(value, true); err != nil {
		*problems = append(*problems, Problem{Field: field, Message: err.Error()})
		return fallback
	}
	return value
}

func readURL(values map[string]string, field, fallback string, allowedSchemes []string, optional bool, problems *[]Problem) string {
	raw, ok := values[field]
	if !ok {
		return fallback
	}
	value := strings.TrimSpace(raw)
	if value == "" {
		if optional {
			return ""
		}
		*problems = append(*problems, Problem{Field: field, Message: "must not be empty"})
		return fallback
	}
	if err := validateURL(value, allowedSchemes); err != nil {
		*problems = append(*problems, Problem{Field: field, Message: err.Error()})
		return fallback
	}
	return value
}

func readURLList(values map[string]string, field string, allowedSchemes []string, problems *[]Problem) []string {
	raw, ok := values[field]
	if !ok || strings.TrimSpace(raw) == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	peers := make([]string, 0, len(parts))
	for _, part := range parts {
		peer := strings.TrimSpace(part)
		if peer == "" {
			*problems = append(*problems, Problem{Field: field, Message: "must not contain empty peer entries"})
			continue
		}
		if err := validateURL(peer, allowedSchemes); err != nil {
			*problems = append(*problems, Problem{Field: field, Message: fmt.Sprintf("%q %s", peer, err.Error())})
			continue
		}
		peers = append(peers, peer)
	}
	return peers
}

func readCommandRuntimeMode(values map[string]string, field string, fallback CommandRuntimeMode, problems *[]Problem) CommandRuntimeMode {
	raw, ok := values[field]
	if !ok {
		return fallback
	}
	value := CommandRuntimeMode(strings.ToLower(strings.TrimSpace(raw)))
	switch value {
	case CommandRuntimeHardwareReadonly, CommandRuntimeSimulator:
		return value
	default:
		*problems = append(*problems, Problem{Field: field, Message: `must be "hardware-readonly" or "simulator"`})
		return fallback
	}
}

func validateURL(value string, allowedSchemes []string) error {
	parsed, err := url.Parse(value)
	if err != nil {
		return fmt.Errorf("must be an absolute URL: %v", err)
	}
	if parsed.Scheme == "" || parsed.Host == "" {
		return errorsForSchemes("must be an absolute URL", allowedSchemes)
	}
	for _, scheme := range allowedSchemes {
		if parsed.Scheme == scheme {
			return nil
		}
	}
	return errorsForSchemes("must use one of the allowed URL schemes", allowedSchemes)
}

func errorsForSchemes(prefix string, allowedSchemes []string) error {
	return fmt.Errorf("%s: %s", prefix, strings.Join(allowedSchemes, ", "))
}

func validateHostPort(value string, allowEmptyHost bool) error {
	host, port, err := net.SplitHostPort(value)
	if err != nil {
		return fmt.Errorf("must be in host:port or :port form: %v", err)
	}
	if !allowEmptyHost && strings.TrimSpace(host) == "" {
		return fmt.Errorf("must include a host before the port")
	}
	if _, err := strconv.Atoi(port); err != nil {
		return fmt.Errorf("port must be numeric")
	}
	portValue, _ := strconv.Atoi(port)
	if portValue <= 0 || portValue > 65535 {
		return fmt.Errorf("port must be from 1 to 65535")
	}
	return nil
}

func hasWhitespace(value string) bool {
	return strings.IndexFunc(value, unicode.IsSpace) >= 0
}

func hasControl(value string) bool {
	return strings.IndexFunc(value, unicode.IsControl) >= 0
}

package commandgate

import "strings"

type RuntimeMode string

const (
	RuntimeModeSimulator RuntimeMode = "simulator"
	RuntimeModeHardware  RuntimeMode = "hardware"

	VerbRequestAutopilotVersion = "request-autopilot-version"
	DefaultSafetyProfile        = "simulator-readback-v1"
)

type PreflightGate struct {
	AllowedSafetyProfiles []string
	AllowedVerbs          []string
	MaxAttempts           int
}

type PreflightRequest struct {
	RuntimeMode              RuntimeMode `json:"runtime_mode"`
	SafetyProfile            string      `json:"safety_profile"`
	TargetEntity             string      `json:"target_entity"`
	Verb                     string      `json:"verb"`
	RequestedBy              string      `json:"requested_by"`
	SenderSystemID           uint8       `json:"sender_system_id"`
	SenderComponentID        uint8       `json:"sender_component_id"`
	Attempts                 int         `json:"attempts"`
	LocalOverride            bool        `json:"local_override"`
	ACKRequired              bool        `json:"ack_required"`
	PostStatePollingRequired bool        `json:"post_state_polling_required"`
	SimulatorConfirmed       bool        `json:"simulator_confirmed"`
	AbortReady               bool        `json:"abort_ready"`
}

type PreflightResult struct {
	Accepted bool              `json:"accepted"`
	Checks   []PreflightCheck  `json:"checks"`
	Evidence PreflightEvidence `json:"evidence"`
}

type PreflightCheck struct {
	Name   string `json:"name"`
	Passed bool   `json:"passed"`
	Detail string `json:"detail"`
}

type PreflightEvidence struct {
	RuntimeMode              RuntimeMode `json:"runtime_mode"`
	SafetyProfile            string      `json:"safety_profile"`
	TargetEntity             string      `json:"target_entity"`
	Verb                     string      `json:"verb"`
	RequestedBy              string      `json:"requested_by"`
	SenderSystemID           uint8       `json:"sender_system_id"`
	SenderComponentID        uint8       `json:"sender_component_id"`
	Attempts                 int         `json:"attempts"`
	MaxAttempts              int         `json:"max_attempts"`
	LocalOverride            bool        `json:"local_override"`
	ACKRequired              bool        `json:"ack_required"`
	PostStatePollingRequired bool        `json:"post_state_polling_required"`
	SimulatorConfirmed       bool        `json:"simulator_confirmed"`
	AbortReady               bool        `json:"abort_ready"`
}

func DefaultSimulatorPreflightGate() PreflightGate {
	return PreflightGate{
		AllowedSafetyProfiles: []string{DefaultSafetyProfile},
		AllowedVerbs:          []string{VerbRequestAutopilotVersion},
		MaxAttempts:           3,
	}
}

func (g PreflightGate) Check(req PreflightRequest) PreflightResult {
	g = g.withDefaults()
	req.RuntimeMode = RuntimeMode(normalize(string(req.RuntimeMode)))
	req.Verb = normalize(req.Verb)
	req.SafetyProfile = normalize(req.SafetyProfile)
	req.TargetEntity = strings.TrimSpace(req.TargetEntity)
	req.RequestedBy = strings.TrimSpace(req.RequestedBy)

	checks := []PreflightCheck{
		check("simulator-only", req.RuntimeMode == RuntimeModeSimulator, "runtime mode must be simulator"),
		check("safety-profile", g.safetyProfileAllowed(req.SafetyProfile), "safety profile must be explicitly allowlisted"),
		check("target-entity", req.TargetEntity != "", "target entity is required"),
		check("requested-by", req.RequestedBy != "", "requesting operator or rule is required"),
		check("stable-sender", req.SenderSystemID != 0 && req.SenderComponentID != 0, "sender system and component ids are required"),
		check("narrow-allowlist", g.verbAllowed(req.Verb), "verb must be explicitly allowlisted"),
		check("bounded-attempts", req.Attempts > 0 && req.Attempts <= g.MaxAttempts, "attempts must be within configured bound"),
		check("local-override", req.LocalOverride, "local override confirmation is required"),
		check("ack-required", req.ACKRequired, "COMMAND_ACK observation must be required"),
		check("post-state-polling", req.PostStatePollingRequired, "post-state polling must be required"),
		check("simulator-confirmed", req.SimulatorConfirmed, "simulator-only confirmation is required"),
		check("abort-ready", req.AbortReady, "abort readiness confirmation is required"),
	}

	return PreflightResult{
		Accepted: allPassed(checks),
		Checks:   checks,
		Evidence: PreflightEvidence{
			RuntimeMode:              req.RuntimeMode,
			SafetyProfile:            req.SafetyProfile,
			TargetEntity:             req.TargetEntity,
			Verb:                     req.Verb,
			RequestedBy:              req.RequestedBy,
			SenderSystemID:           req.SenderSystemID,
			SenderComponentID:        req.SenderComponentID,
			Attempts:                 req.Attempts,
			MaxAttempts:              g.MaxAttempts,
			LocalOverride:            req.LocalOverride,
			ACKRequired:              req.ACKRequired,
			PostStatePollingRequired: req.PostStatePollingRequired,
			SimulatorConfirmed:       req.SimulatorConfirmed,
			AbortReady:               req.AbortReady,
		},
	}
}

func (g PreflightGate) withDefaults() PreflightGate {
	if len(g.AllowedSafetyProfiles) == 0 {
		g.AllowedSafetyProfiles = []string{DefaultSafetyProfile}
	}
	if len(g.AllowedVerbs) == 0 {
		g.AllowedVerbs = []string{VerbRequestAutopilotVersion}
	}
	if g.MaxAttempts <= 0 {
		g.MaxAttempts = 3
	}
	return g
}

func (g PreflightGate) safetyProfileAllowed(profile string) bool {
	if profile == "" {
		return false
	}
	for _, allowed := range g.AllowedSafetyProfiles {
		if normalize(allowed) == profile {
			return true
		}
	}
	return false
}

func (g PreflightGate) verbAllowed(verb string) bool {
	if verb == "" {
		return false
	}
	for _, allowed := range g.AllowedVerbs {
		if normalize(allowed) == verb {
			return true
		}
	}
	return false
}

func check(name string, passed bool, detail string) PreflightCheck {
	return PreflightCheck{Name: name, Passed: passed, Detail: detail}
}

func allPassed(checks []PreflightCheck) bool {
	for _, check := range checks {
		if !check.Passed {
			return false
		}
	}
	return true
}

func normalize(value string) string {
	return strings.ReplaceAll(strings.ToLower(strings.TrimSpace(value)), "_", "-")
}

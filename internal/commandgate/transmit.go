package commandgate

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/c360studio/semlink/internal/mavlink"
)

var (
	ErrPreflightRejected    = errors.New("commandgate: preflight rejected")
	ErrTransmitUnavailable  = errors.New("commandgate: transmit dependency unavailable")
	ErrACKNotAccepted       = errors.New("commandgate: COMMAND_ACK not accepted")
	ErrPostStateNotObserved = errors.New("commandgate: post-state not observed")
)

type MAVLinkTransmitter interface {
	TransmitMAVLink(ctx context.Context, frame []byte) error
}

type CommandACKObserver interface {
	AwaitCommandACK(ctx context.Context, expected ACKExpectation) (ACKEvidence, error)
}

type PostStatePoller interface {
	PollPostState(ctx context.Context, expected PostStateExpectation) (PostStateEvidence, error)
}

type SimulatorTransmitGate struct {
	Preflight       PreflightGate
	Transmitter     MAVLinkTransmitter
	ACKObserver     CommandACKObserver
	PostStatePoller PostStatePoller
	Now             func() time.Time
}

type TransmitRequest struct {
	Preflight          PreflightRequest  `json:"preflight"`
	Sequence           uint8             `json:"sequence"`
	TargetSystemID     uint8             `json:"target_system_id"`
	TargetComponentID  uint8             `json:"target_component_id"`
	RequestedMessageID mavlink.MessageID `json:"requested_message_id"`
}

type TransmitResult struct {
	Accepted      bool                           `json:"accepted"`
	Status        string                         `json:"status"`
	StartedAt     time.Time                      `json:"started_at"`
	Preflight     PreflightResult                `json:"preflight"`
	HardwareBlock *HardwareTransmitBlockEvidence `json:"hardware_block,omitempty"`
	Frames        []FrameEvidence                `json:"frames"`
	ACKs          []ACKEvidence                  `json:"acks"`
	PostState     PostStateEvidence              `json:"post_state"`
}

type FrameEvidence struct {
	Attempt            int               `json:"attempt"`
	Sequence           uint8             `json:"sequence"`
	MessageID          mavlink.MessageID `json:"message_id"`
	CommandID          uint16            `json:"command_id"`
	RequestedMessageID mavlink.MessageID `json:"requested_message_id"`
	SenderSystemID     uint8             `json:"sender_system_id"`
	SenderComponentID  uint8             `json:"sender_component_id"`
	TargetSystemID     uint8             `json:"target_system_id"`
	TargetComponentID  uint8             `json:"target_component_id"`
	Confirmation       uint8             `json:"confirmation"`
	FrameBytes         int               `json:"frame_bytes"`
}

type ACKExpectation struct {
	Attempt           int    `json:"attempt"`
	CommandID         uint16 `json:"command_id"`
	TargetSystemID    uint8  `json:"target_system_id"`
	TargetComponentID uint8  `json:"target_component_id"`
}

type ACKEvidence struct {
	Attempt           int       `json:"attempt"`
	CommandID         uint16    `json:"command_id"`
	Result            uint8     `json:"result"`
	SourceSystemID    uint8     `json:"source_system_id"`
	SourceComponentID uint8     `json:"source_component_id"`
	ObservedAt        time.Time `json:"observed_at"`
}

func (e ACKEvidence) Accepted() bool {
	return e.Result == mavlink.MAVResultAccepted
}

type PostStateExpectation struct {
	CommandID          uint16            `json:"command_id"`
	RequestedMessageID mavlink.MessageID `json:"requested_message_id"`
	TargetEntity       string            `json:"target_entity"`
	TargetSystemID     uint8             `json:"target_system_id"`
	TargetComponentID  uint8             `json:"target_component_id"`
}

type PostStateEvidence struct {
	Observed     bool      `json:"observed"`
	TargetEntity string    `json:"target_entity"`
	Summary      string    `json:"summary"`
	ObservedAt   time.Time `json:"observed_at"`
}

func (g SimulatorTransmitGate) Transmit(ctx context.Context, req TransmitRequest) (TransmitResult, error) {
	startedAt := g.now()
	if RuntimeMode(normalize(string(req.Preflight.RuntimeMode))) == RuntimeModeHardware {
		blocker := DefaultHardwareTransmitBlocker()
		blocker.Now = g.Now
		block, err := blocker.Check(HardwareTransmitRequest{
			RuntimeMode:       req.Preflight.RuntimeMode,
			SafetyProfile:     req.Preflight.SafetyProfile,
			TargetEntity:      req.Preflight.TargetEntity,
			Verb:              req.Preflight.Verb,
			RequestedBy:       req.Preflight.RequestedBy,
			TargetSystemID:    req.TargetSystemID,
			TargetComponentID: req.TargetComponentID,
		})
		return TransmitResult{
			Status:        block.Status,
			StartedAt:     startedAt,
			HardwareBlock: &block,
		}, err
	}

	preflight := g.Preflight.Check(req.Preflight)
	result := TransmitResult{
		Status:    "preflight-rejected",
		StartedAt: startedAt,
		Preflight: preflight,
	}
	if !preflight.Accepted {
		return result, ErrPreflightRejected
	}
	if g.Transmitter == nil || g.ACKObserver == nil || g.PostStatePoller == nil {
		result.Status = "transmit-unavailable"
		return result, ErrTransmitUnavailable
	}
	if req.TargetSystemID == 0 || req.TargetComponentID == 0 {
		result.Status = "invalid-target"
		return result, fmt.Errorf("%w: target system and component ids are required", ErrTransmitUnavailable)
	}

	requestedMessageID := req.RequestedMessageID
	if requestedMessageID == 0 {
		requestedMessageID = mavlink.MessageAutopilotVersion
	}
	attempts := preflight.Evidence.Attempts
	for attempt := 1; attempt <= attempts; attempt++ {
		confirmation := uint8(attempt - 1)
		sequence := req.Sequence + confirmation
		payload := mavlink.RequestMessageCommandPayload(req.TargetSystemID, req.TargetComponentID, requestedMessageID, confirmation)
		frame, err := mavlink.EncodeV2(
			sequence,
			preflight.Evidence.SenderSystemID,
			preflight.Evidence.SenderComponentID,
			mavlink.MessageCommandLong,
			payload,
		)
		if err != nil {
			result.Status = "encode-failed"
			return result, err
		}
		if err := g.Transmitter.TransmitMAVLink(ctx, frame); err != nil {
			result.Status = "transmit-failed"
			return result, err
		}
		result.Frames = append(result.Frames, FrameEvidence{
			Attempt:            attempt,
			Sequence:           sequence,
			MessageID:          mavlink.MessageCommandLong,
			CommandID:          mavlink.MAVCmdRequestMessage,
			RequestedMessageID: requestedMessageID,
			SenderSystemID:     preflight.Evidence.SenderSystemID,
			SenderComponentID:  preflight.Evidence.SenderComponentID,
			TargetSystemID:     req.TargetSystemID,
			TargetComponentID:  req.TargetComponentID,
			Confirmation:       confirmation,
			FrameBytes:         len(frame),
		})

		ack, err := g.ACKObserver.AwaitCommandACK(ctx, ACKExpectation{
			Attempt:           attempt,
			CommandID:         mavlink.MAVCmdRequestMessage,
			TargetSystemID:    req.TargetSystemID,
			TargetComponentID: req.TargetComponentID,
		})
		if err != nil {
			result.Status = "ack-observation-failed"
			return result, err
		}
		if ack.ObservedAt.IsZero() {
			ack.ObservedAt = g.now()
		}
		result.ACKs = append(result.ACKs, ack)
		if ack.CommandID != mavlink.MAVCmdRequestMessage || !ack.Accepted() {
			continue
		}

		postState, err := g.PostStatePoller.PollPostState(ctx, PostStateExpectation{
			CommandID:          mavlink.MAVCmdRequestMessage,
			RequestedMessageID: requestedMessageID,
			TargetEntity:       preflight.Evidence.TargetEntity,
			TargetSystemID:     req.TargetSystemID,
			TargetComponentID:  req.TargetComponentID,
		})
		if err != nil {
			result.Status = "post-state-poll-failed"
			return result, err
		}
		if postState.TargetEntity == "" {
			postState.TargetEntity = preflight.Evidence.TargetEntity
		}
		if postState.ObservedAt.IsZero() {
			postState.ObservedAt = g.now()
		}
		result.PostState = postState
		if !postState.Observed {
			result.Status = "post-state-not-observed"
			return result, ErrPostStateNotObserved
		}
		result.Accepted = true
		result.Status = "accepted"
		return result, nil
	}

	result.Status = "ack-not-accepted"
	return result, ErrACKNotAccepted
}

func (g SimulatorTransmitGate) now() time.Time {
	if g.Now != nil {
		return g.Now()
	}
	return time.Now()
}

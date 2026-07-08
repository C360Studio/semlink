package sitl

import (
	"errors"
	"fmt"

	"github.com/c360studio/semlink/internal/handoff"
)

type Frame string

const (
	FrameRover         Frame = "rover"
	FrameBalanceBot    Frame = "balancebot"
	FrameRoverSkid     Frame = "rover-skid"
	FrameSailboat      Frame = "sailboat"
	FrameSailboatMotor Frame = "sailboat-motor"
)

type ArduPilotLaneConfig struct {
	Vehicle    string
	Frame      Frame
	OutputHost string
	OutputPort int
	Aircraft   string
	Speedup    int
	WipeEEPROM bool
	NoMAVProxy bool
}

type Evidence struct {
	Lane          string
	Vehicle       string
	Frame         string
	MAVLinkOutput string
	Gazebo        bool
	Hardware      bool
}

func DefaultArduRoverLane() ArduPilotLaneConfig {
	return ArduPilotLaneConfig{
		Vehicle:    "Rover",
		Frame:      FrameRover,
		OutputHost: "127.0.0.1",
		OutputPort: 14550,
		Aircraft:   "semlink-ardurover",
		Speedup:    1,
		WipeEEPROM: true,
		NoMAVProxy: true,
	}
}

func ArduRoverLaneFromHandoffProfile(profile handoff.Profile) ArduPilotLaneConfig {
	cfg := DefaultArduRoverLane()
	cfg.OutputHost = profile.MAVLinkUDPHost
	cfg.OutputPort = profile.MAVLinkUDPPort
	return cfg
}

func (c ArduPilotLaneConfig) Validate() error {
	if c.Vehicle != "Rover" {
		return fmt.Errorf("SITL lane supports Rover only, got %q", c.Vehicle)
	}
	if !isRoverFrame(c.Frame) {
		return fmt.Errorf("unsupported Rover frame %q", c.Frame)
	}
	if c.OutputHost == "" {
		return errors.New("MAVLink output host is required")
	}
	if c.OutputPort <= 0 || c.OutputPort > 65535 {
		return fmt.Errorf("MAVLink output port out of range: %d", c.OutputPort)
	}
	return nil
}

func (c ArduPilotLaneConfig) MAVLinkOutput() string {
	return fmt.Sprintf("udpclient:%s:%d", c.OutputHost, c.OutputPort)
}

func (c ArduPilotLaneConfig) SimVehicleArgs() ([]string, error) {
	c = c.withDefaults()
	if err := c.Validate(); err != nil {
		return nil, err
	}
	args := []string{
		"-v", c.Vehicle,
		"-f", string(c.Frame),
	}
	if c.NoMAVProxy {
		args = append(args, "--no-mavproxy")
	}
	args = append(args,
		fmt.Sprintf("--speedup=%d", c.Speedup),
		"--aircraft", c.Aircraft,
		"-A", fmt.Sprintf("--serial0=%s", c.MAVLinkOutput()),
	)
	if c.WipeEEPROM {
		args = append(args, "-w")
	}
	return args, nil
}

func (c ArduPilotLaneConfig) Evidence() (Evidence, error) {
	c = c.withDefaults()
	if err := c.Validate(); err != nil {
		return Evidence{}, err
	}
	return Evidence{
		Lane:          "ardupilot-sitl",
		Vehicle:       c.Vehicle,
		Frame:         string(c.Frame),
		MAVLinkOutput: c.MAVLinkOutput(),
		Gazebo:        false,
		Hardware:      false,
	}, nil
}

func (c ArduPilotLaneConfig) withDefaults() ArduPilotLaneConfig {
	if c.Vehicle == "" {
		c.Vehicle = "Rover"
	}
	if c.Frame == "" {
		c.Frame = FrameRover
	}
	if c.OutputHost == "" {
		c.OutputHost = "127.0.0.1"
	}
	if c.OutputPort == 0 {
		c.OutputPort = 14550
	}
	if c.Aircraft == "" {
		c.Aircraft = "semlink-ardurover"
	}
	if c.Speedup <= 0 {
		c.Speedup = 1
	}
	if !c.NoMAVProxy {
		c.NoMAVProxy = true
	}
	return c
}

func isRoverFrame(frame Frame) bool {
	switch frame {
	case FrameRover, FrameBalanceBot, FrameRoverSkid, FrameSailboat, FrameSailboatMotor:
		return true
	default:
		return false
	}
}

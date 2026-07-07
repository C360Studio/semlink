package main

import (
	"testing"
	"time"

	"github.com/c360studio/semlink/internal/handoff"
)

func TestHandoffProfileValuesPreferRuntimeFlagsForRuntimeFields(t *testing.T) {
	values := handoffProfileValues([]string{
		"SEMLINK_NODE_ID=boat-alpha",
		"SEMLINK_MAVLINK_UDP_LISTEN=:9999",
		"SEMLINK_MESH_PEERS=http://boat-bravo.local:8081",
	}, runtimeProfileValues{
		Listen:                ":8081",
		EmbeddedNATS:          true,
		NATSURL:               "nats://runtime",
		Vehicles:              1,
		Hz:                    5,
		Buffer:                10000,
		MAVLinkUDP:            ":14550",
		CSAPIURL:              "http://127.0.0.1:48080",
		CSAPIInterval:         2 * time.Second,
		CSAPIObservationEvery: 5 * time.Second,
		TAKEnabled:            false,
		TAKMulticast:          "239.2.3.1:6969",
		TAKInterval:           time.Second,
	})

	profile, err := handoff.ParseEnv(values)
	if err != nil {
		t.Fatalf("ParseEnv() error = %v", err)
	}
	if profile.NodeID != "boat-alpha" {
		t.Fatalf("NodeID = %q", profile.NodeID)
	}
	if profile.MAVLinkUDPListen != ":14550" {
		t.Fatalf("MAVLinkUDPListen = %q, want runtime flag value", profile.MAVLinkUDPListen)
	}
	if len(profile.MeshPeers) != 1 || profile.MeshPeers[0] != "http://boat-bravo.local:8081" {
		t.Fatalf("MeshPeers = %#v", profile.MeshPeers)
	}
	if profile.CSAPIURL != "http://127.0.0.1:48080" {
		t.Fatalf("CSAPIURL = %q", profile.CSAPIURL)
	}
}

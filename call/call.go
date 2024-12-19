package call

import (
	"log/slog"
	"os"

	"github.com/CyCoreSystems/ari/v6"
)

var log = slog.New(slog.NewTextHandler(os.Stderr, nil))

func OriginateCall(client ari.Client, bridge *ari.BridgeHandle, endpoint string, callerNumber string) {
	channel, err := client.Channel().Originate(nil, ari.OriginateRequest{
		Endpoint: endpoint,
		App:      "callChildrens",
		CallerID: callerNumber,
	})
	if err != nil {
		log.Error("Failed to originate call", "endpoint", endpoint, "error", err)
		return
	}
	if channel == nil {
		log.Error("Failed to originate call")
		return
	}

	log.Info("Call originated", "endpoint", endpoint, "channelID", channel.ID())
	if err := bridge.AddChannel(channel.ID()); err != nil {
		log.Error("Failed to add new Channel to bridge", "error", err)
		return
	}

	log.Info("Channel added to bridge", "channelID", channel.ID())
}

package bridge

import (
	"log/slog"
	"os"

	"github.com/CyCoreSystems/ari/v6"
)

var log = slog.New(slog.NewTextHandler(os.Stderr, nil))

func CreateBridge(client ari.Client, channelKey *ari.Key) (*ari.BridgeHandle, error) {
	bridge, err := client.Bridge().Create(channelKey, "mixing", "child-call-bridge")
	if err != nil {
		log.Error("Failed to create bridge", "error", err)
		return nil, err
	}
	log.Info("Bridge created", "bridge ID", bridge.ID())
	return bridge, nil
}

func AddChannelToBridge(bridge *ari.BridgeHandle, channelID string) error {
	if err := bridge.AddChannel(channelID); err != nil {
		log.Error("Failed to add channel to bridge", "error", err)
		return err
	}

	log.Info("Channel added to bridge", "channelID", channelID)
	return nil
}

package call

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/CyCoreSystems/ari/v6"
)

var log = slog.New(slog.NewTextHandler(os.Stderr, nil))

func OriginateCall(client ari.Client, bridge *ari.BridgeHandle, endpoint string, callerNumber string, cancelChan chan struct{}, allCancelChans map[string]chan struct{}) (string, error) {
	channel, err := client.Channel().Originate(nil, ari.OriginateRequest{
		Endpoint: endpoint,
		App:      "callChildrens",
		CallerID: callerNumber,
	})
	if err != nil {
		log.Error("Failed to originate call", "endpoint", endpoint, "error", err)
		return "", err
	}
	if channel == nil {
		log.Error("Failed to originate call", "endpoint", endpoint)
		return "", fmt.Errorf("channel is nil for endpoint %s", endpoint)
	}

	log.Info("Call originated", "endpoint", endpoint, "channelID", channel.ID())

	// Monitorar o canal e aguardar ele entrar no estado "Up"
	sub := channel.Subscribe(ari.Events.ChannelStateChange)
	defer sub.Cancel()

	for {
		select {
		case e := <-sub.Events():
			stateChange := e.(*ari.ChannelStateChange)
			log.Info("Channel state changed", "channel", stateChange.Channel.ID, "state", stateChange.Channel.State)
			if stateChange.Channel.State == "Up" {
				log.Info("Channel is Up, adding to bridge", "channelID", channel.ID())

				// Adiciona o canal na bridge
				if err := bridge.AddChannel(channel.ID()); err != nil {
					log.Error("Failed to add new Channel to bridge", "error", err)
					return "", err
				}

				// **Não cancela o canal que atendeu**
				for ep, cancelCh := range allCancelChans {
					if ep != endpoint {
						log.Info("Cancelling call to endpoint", "endpoint", ep)
						close(cancelCh) // Fecha apenas o canal de endpoints que não atenderam
					}
				}
				break
			}
		case <-cancelChan:
			log.Info("Cancelling call to endpoint", "endpoint", endpoint)
			if err := channel.Hangup(); err != nil {
				log.Error("Failed to hangup channel", "error", err)
			}
			return "", nil
		}
	}

	return channel.ID(), nil
}

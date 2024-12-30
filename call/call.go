package call

import (
	"log/slog"
	"os"

	"github.com/CyCoreSystems/ari/v6"
)

var log = slog.New(slog.NewTextHandler(os.Stderr, nil))

func CreateChannel(cl ari.Client, e *ari.StasisStart, endpoint string) *ari.ChannelHandle {
	newChannel, err := cl.Channel().Originate(nil, ari.OriginateRequest{
		Endpoint: "PJSIP/" + endpoint,
		App:      "simple-call",
		CallerID: e.Channel.Caller.Number,
	})
	if err != nil {
		log.Error("Failed to create channel", "error", err)
		return nil
	}
	return newChannel
}

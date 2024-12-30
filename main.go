package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"sync"
	"time"

	"github.com/CyCoreSystems/ari/v6"
	brd "github.com/wbluan/api-stasis-go/bridge"
	"github.com/wbluan/api-stasis-go/call"
	"github.com/wbluan/api-stasis-go/connection"
)

var log = slog.New(slog.NewTextHandler(os.Stderr, nil))

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	fmt.Println("Connecting to ARI")

	cl, err := connection.ConnectToAri()
	if err != nil {
		log.Error("Failed to connect to ARI")
		return
	}

	listenEvents(ctx, cl)
}

func listenEvents(ctx context.Context, cl ari.Client) {
	log.Info("Starting to listen events")
	sub := cl.Bus().Subscribe(nil, "StasisStart")
	defer sub.Cancel()

	for {
		select {
		case e := <-sub.Events():
			slog.Debug("Received event", "event", e)
			stasisStart := e.(*ari.StasisStart)
			if len(stasisStart.Args) == 0 {
				slog.Debug("No args in event. Ignoring event.")
				continue
			}

			ch := cl.Channel().Get(stasisStart.Key(ari.ChannelKey, stasisStart.Channel.ID))
			go handleStasisStartEvent(cl, ch, stasisStart)
			slog.Debug("Event processed", "event", e)

		case <-ctx.Done():
			slog.Info("Context done. Exiting listenEvents")
			return
		}
	}
}

func handleStasisStartEvent(cl ari.Client, ch *ari.ChannelHandle, e *ari.StasisStart) {
	log.Debug("Processing Stasis Start event")

	channels := creteChannels(cl, e)

	// Verify if a channel hangup event is received
	go monitorChannelsHangup(ch, channels)
	// Wait for all channels to reach the target state
	if !waitForChannelState(channels, "Up", 20*time.Second) {
		log.Warn("No channel reached target state", "state", "Up")
		return
	}

	log.Info("All channels are up")
	// Answer the first channel
	if err := answerChannel(ch); err != nil {
		log.Error("Failed to answer channel", "error", err)
		return
	}

	if err := setupBridge(cl, ch, channels); err != nil {
		log.Error("Failed to setup bridge", "error", err)
		return
	}
}

func creteChannels(cl ari.Client, e *ari.StasisStart) []*ari.ChannelHandle {
	var channels []*ari.ChannelHandle
	dialedNumber := e.Args[0]
	switch dialedNumber {
	case "100":
		endpoints := []string{"1101", "1102"}
		for _, endpoint := range endpoints {
			channels = append(channels, call.CreateChannel(cl, e, endpoint))
		}
	default:
		channels = append(channels, call.CreateChannel(cl, e, dialedNumber))
	}
	return channels
}

func answerChannel(ch *ari.ChannelHandle) error {
	slog.Debug("Answering channel", "channel", ch.ID())
	return ch.Answer()
}

func setupBridge(cl ari.Client, ch *ari.ChannelHandle, channels []*ari.ChannelHandle) error {
	bridge, err := brd.CreateBridge(cl, ch.Key())
	if err != nil {
		slog.Error("Failed to create bridge", "error", err)
		return err
	}
	log.Debug("Bridge created", "bridge", bridge.ID())

	if err := brd.AddChannelToBridge(bridge, ch.ID()); err != nil {
		log.Error("Failed to add channel to bridge", "error", err)
		return err
	}

	for _, ch := range channels {
		if err := brd.AddChannelToBridge(bridge, ch.ID()); err != nil {
			log.Error("Failed to add channel to bridge", "error", err)
			return err
		}
	}

	log.Debug("Channels added to bridge", "bridge", bridge.ID())
	return nil
}

func waitForChannelState(channels []*ari.ChannelHandle, targetState string, timeout time.Duration) bool {

	log.Debug("Starting to wait for any channel to reach target state", "targetState", targetState, "timeout", timeout)
	var wg sync.WaitGroup
	result := make(chan bool, len(channels))
	timer := time.NewTimer(timeout)

	defer timer.Stop()

	for _, ch := range channels {
		wg.Add(1)

		go func(ch *ari.ChannelHandle) {
			defer wg.Done()

			stateSub := ch.Subscribe(ari.Events.ChannelStateChange)
			defer stateSub.Cancel()
			for {
				select {
				case <-stateSub.Events():
					data, err := ch.Data()
					if err != nil {
						slog.Error("Failed to get channel data", "error", err)
						continue
					}
					slog.Debug("Channel state changed", "channel", ch.ID(), "state", data.State)
					if data.State == targetState {
						slog.Debug("Channel is in target state", "channel", ch.ID(), "state", targetState)
						result <- true
						return
					}
				case <-timer.C:
					slog.Debug("Timeout waiting for channel state", "channel", ch.ID(), "targetState", targetState)
					return
				}
			}
		}(ch)
	}
	go func() {
		wg.Wait()
		close(result)
	}()

	select {
	case <-result:
		return true
	case <-timer.C:
		return false
	}
}

func monitorChannelsHangup(ch1 *ari.ChannelHandle, channels []*ari.ChannelHandle) {
	channels = append(channels, ch1)
	for i, ch1 := range channels {
		for j, ch2 := range channels {
			// Skip the same channel
			if i == j {
				continue
			}

			hangupEvent := ch1.Subscribe(ari.Events.ChannelHangupRequest)
			if hangupEvent == nil {
				log.Error("Failed to subscribe to hangup events", "channel", ch1.ID())
				continue
			}

			log.Info("Subscribed to hangup event", "channel", ch1.ID())

			go func(ch1, ch2 *ari.ChannelHandle, hangupEvent ari.Subscription) {
				defer hangupEvent.Cancel()
				log.Info("Monitoring hangup event", "channel", ch1.ID(), "linkedChannel", ch2.ID())

				for range hangupEvent.Events() {
					slog.Debug("Channel hangup detected", "channel", ch1.ID())
					err := ch2.Hangup()
					if err != nil {
						log.Error("Failed to hang up linked channel", "linkedChannel", ch2.ID(), "error", err)
						return
					}
					log.Info("Linked channel hung up successfully", "linkedChannel", ch2.ID())
				}
			}(ch1, ch2, hangupEvent)
		}
	}
}

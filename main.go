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

			ch1 := cl.Channel().Get(stasisStart.Key(ari.ChannelKey, stasisStart.Channel.ID))
			go handleStasisStartEvent(cl, ch1, stasisStart)
			slog.Debug("Event processed", "event", e)

		case <-ctx.Done():
			slog.Info("Context done. Exiting listenEvents")
			return
		}
	}
}

func handleStasisStartEvent(cl ari.Client, ch1 *ari.ChannelHandle, e *ari.StasisStart) {
	log.Debug("Processing Stasis Start event")

	var channels []*ari.ChannelHandle
	dialedNumber := e.Args[0]
	switch dialedNumber {
	case "100":
		endpoints := []string{"PJSIP/1101", "PJSIP/1102"}
		for _, endpoint := range endpoints {
			newChannel := call.CreateChannel(cl, e, endpoint)
			channels = append(channels, newChannel)
		}
	default:
		newChannel := call.CreateChannel(cl, e, dialedNumber)
		channels = append(channels, newChannel)
	}

	go monitorChannelsHangup(ch1, channels)

	if !waitForChannelState(channels, "Up", 20*time.Second) {
		log.Warn("No channel reached target state", "state", "Up")
	} else {
		log.Info("All channels are up")
	}

	err := ch1.Answer()
	if err != nil {
		log.Error("Failed to answer channel", "channel", ch1.ID(), "error", err)
		return
	}

	bridge, err := brd.CreateBridge(cl, ch1.Key())
	if err != nil {
		log.Error("Failed to create bridge", "error", err)
		return
	}
	slog.Debug("Bridge created", "bridge", bridge.ID())

	err = brd.AddChannelToBridge(bridge, ch1.ID())
	if err != nil {
		log.Error("Failed to add channel to bridge", "error", err)
		return
	}
	for _, ch := range channels {
		brd.AddChannelToBridge(bridge, ch.ID())
	}

	slog.Debug("Channels added to bridge", "bridge", bridge.ID())
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

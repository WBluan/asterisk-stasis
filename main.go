package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"

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

	fmt.Println("Lestening for new calls")
	sub := cl.Bus().Subscribe(nil, "StasisStart")

	for {
		select {
		case e := <-sub.Events():
			v := e.(*ari.StasisStart)
			fmt.Println("Stasis start", "channel", v.Channel.ID)
			go handleCall(cl, v)
		case <-ctx.Done():
			return
		}
	}
}

func handleCall(cl ari.Client, v *ari.StasisStart) {
	channel := cl.Channel().Get(v.Key(ari.ChannelKey, v.Channel.ID))

	end := channel.Subscribe(ari.Events.StasisEnd)
	defer end.Cancel()

	go func() {
		<-end.Events()
		log.Info("Call ended", "channel", channel.ID())
	}()

	dialedNumber, err := channel.GetVariable("EXTEN")
	if err != nil {
		log.Error("Failed to get dialed number", "error", err)
		return
	}

	callerNumber, err := channel.GetVariable("CALLER_NUMBER")
	if err != nil {
		log.Error("Failed to get caller number", "error", err)
		return
	}

	log.Info("Caller number", "caller", callerNumber)
	log.Info("Dialed number", "dialed", dialedNumber)

	if dialedNumber == "100" {
		bridge, err := brd.CreateBridge(cl, v.Key(ari.ChannelKey, v.Channel.ID))
		if err != nil {
			return
		}
		if err := brd.AddChannelToBridge(bridge, channel.ID()); err != nil {
			return
		}

		endpoints := []string{"pjsip/1103", "pjsip/1102"}
		cancelChans := make(map[string]chan struct{})

		for _, endpoint := range endpoints {
			cancelChans[endpoint] = make(chan struct{})
			go call.OriginateCall(cl, bridge, endpoint, callerNumber, cancelChans[endpoint], cancelChans)
		}
	}
}
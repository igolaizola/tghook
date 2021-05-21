package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"

	"github.com/igolaizola/tghook"
)

func main() {
	// Parse flags
	channel := flag.String("channel", "", "channel name to get messages from")

	flag.Parse()
	if *channel == "" {
		log.Fatal("channel not provided")
	}

	// Create signal based context
	ctx, cancel := context.WithCancel(context.Background())
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, os.Kill)
	go func() {
		select {
		case <-c:
			cancel()
		case <-ctx.Done():
			cancel()
		}
		signal.Stop(c)
	}()

	// Run bot
	if err := tghook.Run(ctx, *channel); err != nil {
		log.Fatal(err)
	}
}

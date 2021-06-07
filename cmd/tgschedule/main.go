package main

import (
	"context"
	"encoding/json"
	"flag"
	"log"
	"os"
	"os/signal"

	"github.com/igolaizola/tghook"
)

func main() {
	// Parse flags
	config := flag.String("config", "", "config file")

	flag.Parse()
	if *config == "" {
		log.Fatal("config not provided")
	}

	data, err := os.ReadFile(*config)
	if err != nil {
		log.Fatalf("couldn't read config file %s: %v", *config, err)
	}
	var schedules []tghook.Schedule
	if err := json.Unmarshal(data, &schedules); err != nil {
		log.Fatalf("couldn't unmarshal json: %v", err)
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
	if err := tghook.RunSchedule(ctx, schedules); err != nil {
		log.Fatal(err)
	}
}

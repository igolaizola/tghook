package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/igolaizola/tghook"
)

func main() {
	// Parse flags
	channel := flag.String("channel", "", "telegram channel to get messages from")
	filter := flag.String("filter", "", "regex filter to search")
	url := flag.String("url", "", "webhook url")
	method := flag.String("method", "GET", "webhook http method (GET, POST...)")
	data := flag.String("data", "", "webhook post data")
	authUser := flag.String("auth-user", "", "basic auth user")
	authPass := flag.String("auth-pass", "", "basic auth pass")
	upper := flag.Bool("upper", false, "convert filtered data to upper case")
	trim := flag.Bool("trim", false, "trim spaces of filtered data")
	wait := flag.Duration("wait", 500*time.Millisecond, "wait time between requests")
	header := make(http.Header)
	headerVal := &headerValue{header: header}
	flag.Var(headerVal, "header", "http header with format header:value")

	flag.Parse()
	if *channel == "" {
		log.Fatal("user not provided")
	}
	if *filter == "" {
		log.Fatal("regex filter not provided")
	}
	if *url == "" {
		log.Fatal("url not provided")
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
	if err := tghook.RunWithHook(ctx, *channel, *wait, *url, *method, *data, *filter, *authUser, *authPass, header, *upper, *trim); err != nil {
		log.Fatal(err)
	}
}

// headerValue is a flags.Value implementation for http.Header
type headerValue struct {
	header http.Header
}

func (h *headerValue) String() string {
	return fmt.Sprintf("%v", h.header)
}

func (h *headerValue) Set(value string) error {
	parts := strings.SplitN(value, ":", 2)
	if len(parts) != 2 {
		return fmt.Errorf("could not parse header value %s", value)
	}
	k, v := strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])
	h.header.Add(k, v)
	return nil
}

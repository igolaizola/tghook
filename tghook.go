package tghook

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

type message struct {
	text string
	time time.Time
	id   int
}

func Run(ctx context.Context, channel string) error {
	msgs, err := messages(channel, 0)
	if err != nil {
		return err
	}
	minID := msgs[len(msgs)-1].id
	for {
		select {
		case <-ctx.Done():
			return nil
		default:
		}
		msgs, err := messages(channel, minID)
		if err != nil {
			log.Println(err)
			continue
		}
		for _, m := range msgs {
			fmt.Println(m.id, m.time, m.text)
		}
		if len(msgs) > 0 {
			minID = msgs[len(msgs)-1].id
		}
	}
}

func messages(channel string, minID int) ([]message, error) {
	u := fmt.Sprintf("https://t.me/s/%s", channel)
	client := &http.Client{
		Timeout: 10 * time.Second,
	}
	r, err := client.Get(u)
	if err != nil {
		return nil, fmt.Errorf("get request failed: %w", err)
	}
	if r.StatusCode != 200 {
		return nil, fmt.Errorf("invalid status code: %s", r.Status)
	}
	defer r.Body.Close()
	doc, err := goquery.NewDocumentFromReader(r.Body)
	if err != nil {
		return nil, fmt.Errorf("couldn't create document: %w", err)
	}

	var msgs []message
	doc.Find(".tgme_widget_message_wrap").Each(func(i int, s *goquery.Selection) {
		var msg message
		s.Find(".tgme_widget_message_text").EachWithBreak(func(i int, s *goquery.Selection) bool {
			msg.text = s.Text()
			return false
		})
		s.Find(".tgme_widget_message_info time").EachWithBreak(func(i int, s *goquery.Selection) bool {
			val, ok := s.Attr("datetime")
			if !ok {
				return true
			}
			val = strings.Split(val, "+")[0]
			var err error
			msg.time, err = time.Parse("2006-01-02T15:04:05", val)
			if err != nil {
				log.Println(fmt.Sprintf("couldn't parse timestamp %s: %v", val, err))
				return true
			}
			return false
		})
		s.Find(".tgme_widget_message").EachWithBreak(func(i int, s *goquery.Selection) bool {
			val, ok := s.Attr("data-post")
			if !ok {
				return true
			}
			val = strings.TrimLeft(val, fmt.Sprintf("%s/", channel))
			var err error
			msg.id, err = strconv.Atoi(val)
			if err != nil {
				log.Println(fmt.Sprintf("couldn't parse int %s: %v", val, err))
				return true
			}
			return false
		})
		if msg.id <= minID {
			return
		}
		msgs = append(msgs, msg)
	})
	return msgs, nil
}

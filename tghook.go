package tghook

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"regexp"
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

func RunWithHook(ctx context.Context, channel string, wait time.Duration, url, method, data, filter, authUser, authPass string, header http.Header, upper, trim bool) error {
	client := &http.Client{
		Timeout: 10 * time.Second,
	}
	return RunWithFilter(ctx, channel, wait, filter, data, upper, trim, func(msg string, t time.Time) {
		if err := webhook(client, url, method, data, authUser, authPass, header); err != nil {
			log.Println(err)
		}
		log.Println("WEBHOOK", msg, t, data)
	})
}

func RunWithFilter(ctx context.Context, channel string, wait time.Duration, filter, data string, upper, trim bool, callback func(string, time.Time)) error {
	re, err := regexp.Compile(filter)
	if err != nil {
		return fmt.Errorf("tghook: couldn't compile regex %s: %w", filter, err)
	}
	return Run(ctx, channel, wait, func(msg string, t time.Time) {
		matches := re.FindStringSubmatch(msg)
		if len(matches) < 1 {
			return
		}
		data := data
		for i, m := range matches[1:] {
			if upper {
				m = strings.ToUpper(m)
			}
			if trim {
				m = strings.TrimSpace(m)
			}
			data = strings.ReplaceAll(data, fmt.Sprintf("$%d", i+1), m)
		}
		callback(data, t)
	})
}

func Run(ctx context.Context, channel string, wait time.Duration, callback func(string, time.Time)) error {
	minID := 0
	first := true
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-time.After(500 * time.Millisecond):
		default:
		}
		msgs, err := messages(channel, minID)
		if err != nil {
			log.Println(err)
			continue
		}
		for _, m := range msgs {
			if m.id > minID {
				minID = m.id
			}
			if first {
				continue
			}
			log.Println("MSG", m.id, m.time, m.text)
			go callback(m.text, m.time)
		}
		first = false
	}
}

func messages(channel string, minID int) ([]message, error) {
	u := fmt.Sprintf("https://t.me/s/%s", channel)
	client := &http.Client{
		Timeout: 10 * time.Second,
	}
	r, err := client.Get(u)
	if err != nil {
		return nil, fmt.Errorf("tghook: get request failed: %w", err)
	}
	if r.StatusCode != 200 {
		return nil, fmt.Errorf("tghook: invalid status code: %s", r.Status)
	}
	defer r.Body.Close()
	doc, err := goquery.NewDocumentFromReader(r.Body)
	if err != nil {
		return nil, fmt.Errorf("tghook: couldn't create document: %w", err)
	}

	var msgs []message
	doc.Find(".tgme_widget_message_wrap").Each(func(i int, s *goquery.Selection) {
		var msg message
		s.Find(".tgme_widget_message_text").EachWithBreak(func(i int, s1 *goquery.Selection) bool {
			msg.text = s1.Text()
			return false
		})
		s.Find(".tgme_widget_message_info time").EachWithBreak(func(i int, s2 *goquery.Selection) bool {
			val, ok := s2.Attr("datetime")
			if !ok {
				return true
			}
			val = strings.Split(val, "+")[0]
			var err error
			msg.time, err = time.Parse("2006-01-02T15:04:05", val)
			if err != nil {
				log.Println(fmt.Sprintf("tghook: couldn't parse timestamp %s: %v", val, err))
				return true
			}
			return false
		})
		s.Find(".tgme_widget_message").EachWithBreak(func(i int, s3 *goquery.Selection) bool {
			val, ok := s3.Attr("data-post")
			if !ok {
				return true
			}
			val = strings.TrimPrefix(val, fmt.Sprintf("%s/", channel))
			var err error
			msg.id, err = strconv.Atoi(val)
			if err != nil {
				log.Println(fmt.Sprintf("tghook: couldn't parse int %s: %v", val, err))
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

func webhook(client *http.Client, url, method, data, authUser, authPass string, header http.Header) error {
	var body io.Reader
	if data != "" {
		body = strings.NewReader(data)
	}
	req, err := http.NewRequest(method, url, body)
	req.Header = header
	if authUser != "" {
		req.SetBasicAuth(authUser, authPass)
	}
	if err != nil {
		return fmt.Errorf("tghook: couldn't create request: %w", err)
	}
	res, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("tghook: request failed: %w", err)
	}
	if res.StatusCode != 200 {
		return fmt.Errorf("tghook: invalid status code: %s", res.Status)
	}
	return nil
}

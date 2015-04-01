package logentries

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"time"

	"github.com/google/go-querystring/query"
	"github.com/vektra/cypress"
)

const cAPIRoot = "https://pull.logentries.com"

type LogentriesRecv struct {
	*http.Client
	RootURL     string
	Options     *EventsOptions
	EventBuffer chan *cypress.Message
}

type EventsOptions struct {
	Start  int    `url:"start,omitempty"`
	End    int    `url:"end,omitempty"`
	Filter string `url:"filter,omitempty"`
	Limit  int    `url:"limit,omitempty"`
}

type EventsResponse struct {
	Response string `json:"response"`
	Reason   string `json:"reason"`
	Events   []*cypress.Message
}

func NewLogentriesRecv(key, host, log string, options *EventsOptions, bufferSize int) (*LogentriesRecv, error) {
	root := fmt.Sprintf("%s/%s/hosts/%s/%s/", cAPIRoot, key, host, log)

	url, err := url.Parse(root)
	if err != nil {
		return nil, err
	}

	return &LogentriesRecv{
		Client:      &http.Client{},
		RootURL:     url.String(),
		Options:     options,
		EventBuffer: make(chan *cypress.Message, bufferSize),
	}, nil
}

func (lr *LogentriesRecv) SetDefaultOptions(o *EventsOptions) *EventsOptions {
	if o.Start == 0 {
		o.Start = lr.Options.Start
	}
	if o.End == 0 {
		o.End = lr.Options.End
	}
	if o.Filter == "" {
		o.Filter = lr.Options.Filter
	}
	if o.Limit == 0 {
		o.Limit = lr.Options.Limit
	}

	return o
}

func (lr *LogentriesRecv) EncodeURL(o *EventsOptions) string {
	url := lr.RootURL

	v, _ := query.Values(o)
	if q := v.Encode(); q != "" {
		url = url + "?" + q
	}

	return url
}

func (lr *LogentriesRecv) GetBody(url string) ([]byte, error) {
	req, err := http.NewRequest("GET", url, nil)

	resp, err := lr.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return body, err
}

func NewEvents(body []byte) ([]*cypress.Message, error) {
	var events EventsResponse
	err := json.Unmarshal(body, &events)

	if err == nil {
		if events.Response == "error" {
			message := fmt.Sprintf("Logentries error: %s", events.Response, events.Reason)
			return nil, errors.New(message)
		} else {
			// Ok but no events
			return nil, errors.New("Logentires error: No events")
		}

	} else {
		// Log lines sent back verbatim, not proper JSON
		logs := bytes.Split(body, []byte("\n"))

		var events []*cypress.Message

		for _, log := range logs {
			if string(log) != "" {
				var message cypress.Message

				err = json.Unmarshal(log, &message)
				if err != nil {
					message = *cypress.Log()
					message.AddString("message", string(log))
				}

				events = append(events, &message)
			}
		}

		return events, nil
	}

	return nil, nil
}

func (lr *LogentriesRecv) Search(o *EventsOptions) ([]*cypress.Message, error) {
	opts := lr.SetDefaultOptions(o)
	url := lr.EncodeURL(opts)

	body, err := lr.GetBody(url)
	if err != nil {
		return nil, err
	}

	return NewEvents(body)
}

func milliseconds(t time.Time) int {
	nanos := t.UnixNano()
	millis := nanos / 1000000
	return int(millis)
}

func (lr *LogentriesRecv) BufferEvents(events []*cypress.Message) error {
	for _, event := range events {
		select {

		case lr.EventBuffer <- event:
			lr.Options.Start = milliseconds(event.GetTimestamp().Time())

		default:
			break
		}
	}

	return nil
}

func (lr *LogentriesRecv) Generate() (*cypress.Message, error) {
	select {

	case event := <-lr.EventBuffer:
		return event, nil

	case <-time.After(time.Second * 1):
		return nil, nil

	default:
		events, err := lr.Search(lr.Options)
		if err != nil {
			return nil, err
		}

		lr.BufferEvents(events)

		return lr.Generate()
	}
}

func (lr *LogentriesRecv) Close() error {
	close(lr.EventBuffer)
	return nil
}

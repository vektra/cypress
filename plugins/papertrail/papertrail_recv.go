package papertrail

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/google/go-querystring/query"
	"github.com/vektra/cypress"
)

const cAPIRoot = "https://papertrailapp.com/pr/v1/events/search.json"

type PapertrailRecv struct {
	*http.Client
	Token       string
	Options     *EventsOptions
	EventBuffer chan *Event
}

type EventsOptions struct {
	Q        string `url:"q,omitempty"`
	GroupId  string `url:"group_id,omitempty"`
	SystemId string `url:"system_id,omitempty"`
	MinId    string `url:"min_id,omitempty"`
	MaxId    string `url:"max_id,omitempty"`
	MinTime  string `url:"min_time,omitempty"`
	MaxTime  string `url:"max_time,omitempty"`
	Tail     bool   `url:"tail,omitempty"`
}

type EventsResponse struct {
	Events           []*Event `json:"events"`
	MinId            string   `json:"min_id"`
	MaxId            string   `json:"max_id"`
	ReachedBeginning bool     `json:"reached_beginning"`
	ReachedTimeLimit bool     `json:"reached_time_limit"`
}

type Event struct {
	Id                string `json:"id'`
	ReceivedAt        string `json:"received_at"`
	DisplayReceivedAt string `json:"display_received_at"`
	SourceName        string `json:"source_name"`
	SourceId          uint32 `json:"source_id"`
	SourceIp          string `json:"source_ip"`
	Facility          string `json:"facility"`
	Severity          string `json:"severity"`
	Program           string `json:"program"`
	Message           string `json:"message"`
}

func NewPapertrailRecv(token string, options *EventsOptions, bufferSize int) *PapertrailRecv {
	return &PapertrailRecv{
		Client:      &http.Client{},
		Token:       token,
		Options:     options,
		EventBuffer: make(chan *Event, bufferSize),
	}
}

func (pr *PapertrailRecv) SetDefaultOptions(o *EventsOptions) *EventsOptions {
	if o.Q == "" {
		o.Q = pr.Options.Q
	}
	if o.GroupId == "" {
		o.GroupId = pr.Options.GroupId
	}
	if o.SystemId == "" {
		o.SystemId = pr.Options.SystemId
	}
	if o.MinId == "" {
		o.MinId = pr.Options.MinId
	}
	if o.MaxId == "" {
		o.MaxId = pr.Options.MaxId
	}
	if o.MinTime == "" {
		o.MinTime = pr.Options.MinTime
	}
	if o.MaxTime == "" {
		o.MaxTime = pr.Options.MaxTime
	}

	return o
}

func (pr *PapertrailRecv) EncodeURL(o *EventsOptions) string {
	url := cAPIRoot

	v, _ := query.Values(o)
	if q := v.Encode(); q != "" {
		url = url + "?" + q
	}

	return url
}

func (pr *PapertrailRecv) GetBody(url string) ([]byte, error) {
	req, err := http.NewRequest("GET", url, nil)
	req.Header.Add("X-Papertrail-Token", pr.Token)

	resp, err := pr.Do(req)
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

func NewEvents(body []byte) ([]*Event, error) {
	var events EventsResponse

	err := json.Unmarshal(body, &events)
	if err != nil {
		return nil, err
	}

	return events.Events, nil
}

func (pr *PapertrailRecv) Search(o *EventsOptions) ([]*Event, error) {
	opts := pr.SetDefaultOptions(o)
	url := pr.EncodeURL(opts)

	body, err := pr.GetBody(url)
	if err != nil {
		return nil, err
	}

	return NewEvents(body)
}

func NewCypressMessage(event *Event) (*cypress.Message, error) {
	var message cypress.Message

	err := json.Unmarshal([]byte(event.Message), &message)
	if err != nil {
		message = *cypress.Log()
		message.Add("message", event.Message)
	}

	return &message, nil
}

func (pr *PapertrailRecv) BufferEvents(events []*Event) error {
	for _, event := range events {
		select {

		case pr.EventBuffer <- event:
			pr.Options.MinId = event.Id

		default:
			break
		}
	}

	return nil
}

func (pr *PapertrailRecv) Generate() (*cypress.Message, error) {
	select {

	case event := <-pr.EventBuffer:
		return NewCypressMessage(event)

	case <-time.After(time.Second * 1):
		return nil, nil

	default:
		events, err := pr.Search(pr.Options)
		if err != nil {
			return nil, err
		}

		pr.BufferEvents(events)

		return pr.Generate()
	}
}

func (pr *PapertrailRecv) Close() error {
	close(pr.EventBuffer)
	return nil
}

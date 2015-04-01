package loggly

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"time"

	"github.com/google/go-querystring/query"
	"github.com/vektra/cypress"
)

const cAPIRootRSID = "loggly.com/lrv2/search"
const cAPIRootEvents = "loggly.com/lrv2/events"

type LogglyRecv struct {
	*http.Client
	Username      string
	Password      string
	RSIDRootURL   string
	EventsRootURL string
	RSIDOptions   *RSIDOptions
	EventsOptions *EventsOptions
	EventBuffer   chan *Event
}

type RSIDOptions struct {
	Q     string `url:"q,omitempty"`
	From  string `url:"from,omitempty"`
	Until string `url:"until,omitempty"`
	Order string `url:"order,omitempty"`
	Size  uint   `url:"size,omitempty"`
}

type RSIDResponse struct {
	RSID `json:"rsid"`
}

type RSID struct {
	Status      string  `json:"status"`
	DateFrom    uint    `json:"date_from"`
	ElapsedTime float64 `json:"elapsed_time"`
	DateTo      uint    `json:"date_to"`
	ID          string  `json:"id"`
}

type EventsOptions struct {
	RSID    string `url:"rsid"`
	Page    uint   `url:"page,omitempty"`
	Format  string `url:"format,omitempty"`
	Columns string `url:"columns,omitempty"`
}

type EventsResponse struct {
	TotalEvents uint     `json:"total_events"`
	Page        uint     `json:"page"`
	Events      []*Event `json:"events"`
}

type Event struct {
	Timestamp uint   `json:"timestamp"`
	Logmsg    string `json:"logmsg"`
}

func NewLogglyRecv(account, username, password string, ro *RSIDOptions, eo *EventsOptions, bufferSize int) (*LogglyRecv, error) {
	rsid := fmt.Sprintf("http://%s.%s", account, cAPIRootRSID)
	rsidUrl, err := url.Parse(rsid)
	if err != nil {
		return nil, err
	}

	events := fmt.Sprintf("http://%s.%s", account, cAPIRootEvents)
	eventsUrl, err := url.Parse(events)
	if err != nil {
		return nil, err
	}

	return &LogglyRecv{
		Client:        &http.Client{},
		Username:      username,
		Password:      password,
		RSIDRootURL:   rsidUrl.String(),
		EventsRootURL: eventsUrl.String(),
		RSIDOptions:   ro,
		EventsOptions: eo,
		EventBuffer:   make(chan *Event, bufferSize),
	}, nil
}

func (lr *LogglyRecv) SetDefaultRSIDOptions(o *RSIDOptions) *RSIDOptions {
	if o.Q == "" {
		o.Q = lr.RSIDOptions.Q
	}
	if o.From == "" {
		o.From = lr.RSIDOptions.From
	}
	if o.Until == "" {
		o.Until = lr.RSIDOptions.Until
	}
	if o.Order == "" {
		o.Order = lr.RSIDOptions.Order
	}
	if o.Size == 0 {
		o.Size = lr.RSIDOptions.Size
	}

	return o
}

func (lr *LogglyRecv) SetDefaultEventsOptions(o *EventsOptions) *EventsOptions {
	if o.RSID == "" {
		o.RSID = lr.EventsOptions.RSID
	}
	if o.Page == 0 {
		o.Page = lr.EventsOptions.Page
	}
	if o.Format == "" {
		o.Format = lr.EventsOptions.Format
	}
	if o.Columns == "" {
		o.Columns = lr.EventsOptions.Columns
	}

	return o
}

func (lr *LogglyRecv) EncodeRSIDURL(o *RSIDOptions) string {
	url := lr.RSIDRootURL

	v, _ := query.Values(o)
	if q := v.Encode(); q != "" {
		url = url + "?" + q
	}

	return url
}

func (lr *LogglyRecv) EncodeEventsURL(o *EventsOptions) string {
	url := lr.EventsRootURL

	v, _ := query.Values(o)
	if q := v.Encode(); q != "" {
		url = url + "?" + q
	}

	return url
}

func (lr *LogglyRecv) Search(ro *RSIDOptions, eo *EventsOptions) ([]*Event, error) {
	rsid, err := lr.SearchRSID(ro)
	if err != nil {
		return nil, err
	}

	eo.RSID = rsid.ID

	events, err := lr.SearchEvents(eo)
	if err != nil {
		return nil, err
	}

	return events, nil
}

func (lr *LogglyRecv) GetBody(url string) ([]byte, error) {
	req, err := http.NewRequest("GET", url, nil)
	req.SetBasicAuth(lr.Username, lr.Password)

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

func NewRSID(body []byte) (*RSIDResponse, error) {
	var rsid RSIDResponse

	err := json.Unmarshal(body, &rsid)
	if err != nil {
		return nil, err
	}

	return &rsid, nil
}

func (lr *LogglyRecv) SearchRSID(o *RSIDOptions) (*RSIDResponse, error) {
	opts := lr.SetDefaultRSIDOptions(o)
	url := lr.EncodeRSIDURL(opts)

	body, err := lr.GetBody(url)
	if err != nil {
		return nil, err
	}

	return NewRSID(body)
}

func NewEvents(body []byte) ([]*Event, error) {
	var events EventsResponse

	err := json.Unmarshal(body, &events)
	if err != nil {
		return nil, err
	}

	return events.Events, nil
}

func (lr *LogglyRecv) SearchEvents(o *EventsOptions) ([]*Event, error) {
	opts := lr.SetDefaultEventsOptions(o)
	url := lr.EncodeEventsURL(opts)

	body, err := lr.GetBody(url)
	if err != nil {
		return nil, err
	}

	return NewEvents(body)
}

func NewCypressMessage(event *Event) (*cypress.Message, error) {
	var message cypress.Message

	err := json.Unmarshal([]byte(event.Logmsg), &message)
	if err != nil {
		message = *cypress.Log()
		message.Add("message", event.Logmsg)
	}

	return &message, nil
}

func (lr *LogglyRecv) BufferEvents(events []*Event) error {
	for _, event := range events {
		select {

		case lr.EventBuffer <- event:
			continue

		default:
			break
		}
	}
	lr.EventsOptions.Page = lr.EventsOptions.Page + 1

	return nil
}

func (lr *LogglyRecv) Generate() (*cypress.Message, error) {
	select {

	case event := <-lr.EventBuffer:
		return NewCypressMessage(event)

	case <-time.After(time.Second * 1):
		return nil, nil

	default:
		events, err := lr.Search(lr.RSIDOptions, lr.EventsOptions)
		if err != nil {
			return nil, err
		}

		lr.BufferEvents(events)

		return lr.Generate()
	}
}

func (lr *LogglyRecv) Close() error {
	close(lr.EventBuffer)
	return nil
}

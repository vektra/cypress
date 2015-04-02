package papertrail

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/vektra/cypress/plugins/lib/tcplog"
	"github.com/vektra/neko"
)

// For online test
const cEndpoint = "TEST_PAPERTRAIL_URL"
const cSSL = "TEST_PAPERTRAIL_SSL"
const cToken = "TEST_PAPERTRAIL_TOKEN"

func TestSetDefaultOptions(t *testing.T) {
	n := neko.Start(t)

	defaults := &EventsOptions{
		Q:        "query",
		GroupId:  "123456",
		SystemId: "654321",
		MinId:    "min id",
		MaxId:    "max id",
		MinTime:  "24hr",
		MaxTime:  "1hr"}

	pr := NewPapertrailRecv("papertrail-token", defaults, 100)

	n.It("sets default when option is blank", func() {
		options := &EventsOptions{}

		actual := pr.SetDefaultOptions(options)

		require.Equal(t, defaults, actual)
	})

	n.It("does not set default when option is present", func() {
		options := &EventsOptions{
			Q:        "better query",
			GroupId:  "234567",
			SystemId: "765432",
			MinId:    "minner id",
			MaxId:    "maxxer id",
			MinTime:  "72hr",
			MaxTime:  "24hr"}

		actual := pr.SetDefaultOptions(options)

		require.Equal(t, options, actual)
	})

	n.Meow()
}

func TestEncodeURL(t *testing.T) {
	n := neko.Start(t)

	pr := NewPapertrailRecv("papertrail-token", &EventsOptions{}, 100)

	n.It("is root url if options are empty", func() {
		actual := pr.EncodeURL(&EventsOptions{})

		expected := cAPIRoot

		require.Equal(t, expected, actual)
	})

	n.It("is root url + options if options are not empty", func() {
		actual := pr.EncodeURL(&EventsOptions{
			Q:        "query",
			GroupId:  "123456",
			SystemId: "654321",
			MinId:    "min id",
			MaxId:    "max id",
			MinTime:  "24hr",
			MaxTime:  "1hr"})

		expected := cAPIRoot + "?group_id=123456&max_id=max+id&max_time=1hr&min_id=min+id&min_time=24hr&q=query&system_id=654321"

		require.Equal(t, expected, actual)
	})

	n.Meow()
}

func TestNewEvents(t *testing.T) {
	n := neko.Start(t)

	n.It("unmarshals events response and returns events", func() {
		body := []byte(`
{
   "min_id":"519436550648672265",
   "max_id":"519436550648672265",
   "events":[
      {
         "id":"519436550648672265",
         "source_ip":"208.90.212.182",
         "program":"logger",
         "message":"{\"@timestamp\":\"2015-03-31T05:42:27.323399324Z\",\"@type\":\"log\",\"@version\":\"1\",\"message\":\"awesome\"}",
         "received_at":"2015-03-30T22:42:26-07:00",
         "generated_at":"2015-03-30T22:42:26-07:00",
         "display_received_at":"Mar 30 22:42:26",
         "source_id":78261334,
         "source_name":"208.90.212.182",
         "hostname":"208.90.212.182",
         "severity":"Emergency",
         "facility":"Kernel"
      }
   ],
   "reached_beginning":true,
   "min_time_at":"2015-03-30T21:09:13-07:00"
}
		`)

		events, err := NewEvents(body)

		require.NoError(t, err)
		require.Equal(t, 1, len(events))

		expected := "{\"@timestamp\":\"2015-03-31T05:42:27.323399324Z\",\"@type\":\"log\",\"@version\":\"1\",\"message\":\"awesome\"}"
		actual := events[0].Message

		require.Equal(t, expected, actual)
	})

	n.Meow()
}

func TestNewCypressMessage(t *testing.T) {
	n := neko.Start(t)

	n.It("unmarshals valid json to cypress message", func() {
		event := &Event{Message: "{\"@timestamp\":\"2015-03-31T05:42:27.323399324Z\",\"@type\":\"log\",\"@version\":\"1\",\"message\":\"awesome\"}"}

		message, err := NewCypressMessage(event)
		require.NoError(t, err)

		msg, ok := message.GetString("message")
		require.True(t, ok)

		require.Equal(t, "awesome", msg)

		timestamp, _ := time.Parse(time.RFC3339Nano, "2015-03-31T05:42:27.323399324Z")

		require.Equal(t, timestamp, message.GetTimestamp().Time())
		require.Equal(t, "log", message.StringType())
		require.Equal(t, 1, message.GetVersion())
	})

	n.It("creates new cypress message from invalid json", func() {
		line := "> 2015-03-31T05:42:27.323399324Z * @type=log @version=1 message=awesome"
		event := &Event{Message: line}

		message, err := NewCypressMessage(event)
		require.NoError(t, err)

		msg, ok := message.GetString("message")
		require.True(t, ok)

		require.Equal(t, line, msg)

		timestamp, _ := time.Parse(time.RFC3339Nano, "2015-03-31T05:42:27.323399324Z")

		require.NotEqual(t, timestamp, message.GetTimestamp().Time())
		require.Equal(t, "log", message.StringType())
		require.Equal(t, 1, message.GetVersion())
	})

	n.Meow()
}

func TestBufferEvents(t *testing.T) {
	n := neko.Start(t)

	n.It("adds event to events buffer", func() {
		pr := NewPapertrailRecv("papertrail-token", &EventsOptions{}, 100)

		var events []*Event
		expected := &Event{Message: "radical"}
		events = append(events, expected)

		err := pr.BufferEvents(events)
		require.NoError(t, err)

		actual := <-pr.EventBuffer

		require.Equal(t, expected, actual)
	})

	n.It("sets min id to be the id from the added event", func() {
		pr := NewPapertrailRecv("papertrail-token", &EventsOptions{}, 100)

		var events []*Event
		expected := &Event{Message: "radical", Id: "45029"}
		events = append(events, expected)

		err := pr.BufferEvents(events)
		require.NoError(t, err)

		require.Equal(t, expected.Id, pr.Options.MinId)
	})

	n.It("does not wait on full buffer", func() {
		pr := NewPapertrailRecv("papertrail-token", &EventsOptions{}, 1)

		var events []*Event
		expected := &Event{Message: "radical", Id: "45029"}
		extra := &Event{Message: "tubular", Id: "45030"}
		events = append(events, expected)
		events = append(events, extra)

		err := pr.BufferEvents(events)
		require.NoError(t, err)

		require.Equal(t, expected.Id, pr.Options.MinId)
	})

	n.Meow()
}

func TestPapertrailOnline(t *testing.T) {
	endpoint := os.Getenv(cEndpoint)
	if endpoint == "" {
		t.Skipf("%s is not set.", cEndpoint)
	}

	ssl := os.Getenv(cSSL)
	if ssl == "" {
		ssl = "true"
	}

	token := os.Getenv(cToken)
	if token == "" {
		t.Skipf("%s is not set.", cToken)
	}

	// Send message to papertrail

	l := NewLogger(endpoint, ssl == "true")
	go l.Run()

	expected := tcplog.NewMessage(t)
	l.Receive(expected)

	time.Sleep(10 * time.Second)

	// Read back message from papertrail

	q, _ := expected.GetString("message")
	options := &EventsOptions{Q: q, Tail: false}
	pr := NewPapertrailRecv(token, options, 100)

	actual, err := pr.Generate()
	require.NoError(t, err)

	// Make sure its the same message

	require.Equal(t, expected.GetTimestamp().Time(), actual.GetTimestamp().Time())
	require.Equal(t, expected.GetVersion(), actual.GetVersion())
	require.Equal(t, expected.GetSessionId(), actual.GetSessionId())
	require.Equal(t, expected.GetTags(), actual.GetTags())

	expectedMessage, _ := expected.GetString("message")
	actualMessage, _ := actual.GetString("message")

	require.Equal(t, expectedMessage, actualMessage)
}

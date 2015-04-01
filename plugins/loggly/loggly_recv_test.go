package loggly

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/vektra/cypress/plugins/lib/tcplog"
	"github.com/vektra/neko"
)

// For online test
const cAccount = "TEST_LOGGLY_ACCOUNT"
const cUsername = "TEST_LOGGLY_USERNAME"
const cPassword = "TEST_LOGGLY_PASSWORD"

func TestSetDefaultRSIDOptions(t *testing.T) {
	n := neko.Start(t)

	rsid := &RSIDOptions{
		Q:     "query",
		From:  "24hr",
		Until: "1hr",
		Order: "asc",
		Size:  100}
	events := &EventsOptions{}

	lr, _ := NewLogglyRecv("account", "username", "password", rsid, events, 100)

	n.It("sets default when option is blank", func() {
		options := &RSIDOptions{}

		actual := lr.SetDefaultRSIDOptions(options)

		require.Equal(t, rsid, actual)
	})

	n.It("does not set default when option is present", func() {
		options := &RSIDOptions{
			Q:     "query",
			From:  "24hr",
			Until: "1hr",
			Order: "desc",
			Size:  50}

		actual := lr.SetDefaultRSIDOptions(options)

		require.Equal(t, options, actual)
	})

	n.Meow()
}

func TestSetDefaultEventsOptions(t *testing.T) {
	n := neko.Start(t)

	rsid := &RSIDOptions{}
	events := &EventsOptions{
		RSID:    "123456",
		Page:    0,
		Format:  "json",
		Columns: "100"}

	lr, _ := NewLogglyRecv("account", "username", "password", rsid, events, 100)

	n.It("sets default when option is blank", func() {
		options := &EventsOptions{}

		actual := lr.SetDefaultEventsOptions(options)

		require.Equal(t, events, actual)
	})

	n.It("does not set default when option is present", func() {
		options := &EventsOptions{
			RSID:    "234567",
			Page:    3,
			Format:  "json",
			Columns: "50"}

		actual := lr.SetDefaultEventsOptions(options)

		require.Equal(t, options, actual)
	})

	n.Meow()
}

func TestEncodeRSIDURL(t *testing.T) {
	n := neko.Start(t)

	lr, _ := NewLogglyRecv("account", "username", "password", &RSIDOptions{}, &EventsOptions{}, 100)

	n.It("is root url if options are empty", func() {
		actual := lr.EncodeRSIDURL(&RSIDOptions{})

		expected := "http://account.loggly.com/lrv2/search"

		require.Equal(t, expected, actual)
	})

	n.It("is root url + options if options are not empty", func() {
		actual := lr.EncodeRSIDURL(&RSIDOptions{
			Q:     "query",
			From:  "24hr",
			Until: "1hr",
			Order: "asc",
			Size:  100})

		expected := "http://account.loggly.com/lrv2/search?from=24hr&order=asc&q=query&size=100&until=1hr"

		require.Equal(t, expected, actual)
	})

	n.Meow()
}

func TestEncodeEventsURL(t *testing.T) {
	n := neko.Start(t)

	lr, _ := NewLogglyRecv("account", "username", "password", &RSIDOptions{}, &EventsOptions{}, 100)

	n.It("is root url if options are empty", func() {
		actual := lr.EncodeEventsURL(&EventsOptions{RSID: "123456"})

		expected := "http://account.loggly.com/lrv2/events?rsid=123456"

		require.Equal(t, expected, actual)
	})

	n.It("is root url + options if options are not empty", func() {
		actual := lr.EncodeEventsURL(&EventsOptions{
			RSID:    "234567",
			Page:    3,
			Format:  "json",
			Columns: "50"})

		expected := "http://account.loggly.com/lrv2/events?columns=50&format=json&page=3&rsid=234567"

		require.Equal(t, expected, actual)
	})

	n.Meow()
}

func TestNewEvents(t *testing.T) {
	n := neko.Start(t)

	n.It("unmarshals events response and returns events", func() {
		body := []byte(`
{
  "total_events": 33,
  "page": 0,
  "events": [
    {
      "tags": [],
      "timestamp": 1427835412204,
			"logmsg": "{\"@timestamp\":\"2015-03-31T05:42:27.323399324Z\",\"@type\":\"log\",\"@version\":\"1\",\"message\":\"awesome\"}",
      "logtypes": [
        "json"
      ],
      "id": "74ed3e2a-d7e8-11e4-8010-0e2b0be35c37"
    }
  ]
}
		`)

		events, err := NewEvents(body)

		require.NoError(t, err)
		require.Equal(t, 1, len(events))

		expected := "{\"@timestamp\":\"2015-03-31T05:42:27.323399324Z\",\"@type\":\"log\",\"@version\":\"1\",\"message\":\"awesome\"}"
		actual := events[0].Logmsg

		require.Equal(t, expected, actual)
	})

	n.Meow()
}

func TestNewCypressMessage(t *testing.T) {
	n := neko.Start(t)

	n.It("unmarshals valid json to cypress message", func() {
		event := &Event{Logmsg: "{\"@timestamp\":\"2015-03-31T05:42:27.323399324Z\",\"@type\":\"log\",\"@version\":\"1\",\"message\":\"awesome\"}"}

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
		event := &Event{Logmsg: line}

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
		lr, _ := NewLogglyRecv("account", "username", "password", &RSIDOptions{}, &EventsOptions{}, 100)

		var events []*Event
		expected := &Event{Logmsg: "radical"}
		events = append(events, expected)

		err := lr.BufferEvents(events)
		require.NoError(t, err)

		actual := <-lr.EventBuffer

		require.Equal(t, expected, actual)
	})

	n.It("sets page to be incremented when all events added", func() {
		lr, _ := NewLogglyRecv("account", "username", "password", &RSIDOptions{}, &EventsOptions{}, 100)

		var events []*Event
		expected := &Event{Logmsg: "radical"}
		events = append(events, expected)

		err := lr.BufferEvents(events)
		require.NoError(t, err)

		require.Equal(t, uint(1), lr.EventsOptions.Page)
	})

	n.It("does not wait on full buffer", func() {
		lr, _ := NewLogglyRecv("account", "username", "password", &RSIDOptions{}, &EventsOptions{}, 1)

		var events []*Event
		expected := &Event{Logmsg: "radical"}
		extra := &Event{Logmsg: "tubular"}
		events = append(events, expected)
		events = append(events, extra)

		err := lr.BufferEvents(events)
		require.NoError(t, err)

		require.Equal(t, uint(1), lr.EventsOptions.Page)
	})

	n.Meow()
}

func TestLogglyOnline(t *testing.T) {
	token := os.Getenv(cToken)
	if token == "" {
		t.Skipf("%s is not set.", cToken)
	}

	account := os.Getenv(cAccount)
	if account == "" {
		t.Skipf("%s is not set.", cAccount)
	}

	username := os.Getenv(cUsername)
	if username == "" {
		t.Skipf("%s is not set.", cUsername)
	}

	password := os.Getenv(cPassword)
	if password == "" {
		t.Skipf("%s is not set.", cPassword)
	}

	// Send message to loggly

	l := NewLogger(token)

	expected := tcplog.NewMessage(t)
	l.Receive(expected)

	time.Sleep(20 * time.Second)

	// Read back message from loggly

	ro := &RSIDOptions{Size: 1}
	eo := &EventsOptions{}
	lr, err := NewLogglyRecv(account, username, password, ro, eo, 100)
	require.NoError(t, err)

	actual, err := lr.Generate()
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

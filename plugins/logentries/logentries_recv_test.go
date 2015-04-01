package logentries

import (
	"encoding/json"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/vektra/cypress"
	"github.com/vektra/cypress/plugins/lib/tcplog"
	"github.com/vektra/neko"
)

// For online test
const cAccountKey = "TEST_LOGENTRIES_ACCOUNT_KEY"
const cHost = "TEST_LOGENTRIES_HOST"
const cLog = "TEST_LOGENTRIES_LOG"

func TestSetDefaultOptions(t *testing.T) {
	n := neko.Start(t)

	defaults := &EventsOptions{
		Start:  milliseconds(time.Now()),
		End:    milliseconds(time.Now()),
		Filter: "query",
		Limit:  100}

	lr, _ := NewLogentriesRecv("key", "host", "log", defaults, 100)

	n.It("sets default when option is blank", func() {
		options := &EventsOptions{}

		actual := lr.SetDefaultOptions(options)

		require.Equal(t, defaults, actual)
	})

	n.It("does not set default when option is present", func() {
		options := &EventsOptions{
			Start:  milliseconds(time.Now()) + 20,
			End:    milliseconds(time.Now()) - 20,
			Filter: "filter",
			Limit:  50}

		actual := lr.SetDefaultOptions(options)

		require.Equal(t, options, actual)
	})

	n.Meow()
}

func TestEncodeURL(t *testing.T) {
	n := neko.Start(t)

	lr, _ := NewLogentriesRecv("key", "host", "log", &EventsOptions{}, 100)

	n.It("is root url if options are empty", func() {
		actual := lr.EncodeURL(&EventsOptions{})

		expected := "https://pull.logentries.com/key/hosts/host/log/"

		require.Equal(t, expected, actual)
	})

	n.It("is root url + options if options are not empty", func() {
		now := milliseconds(time.Now())

		actual := lr.EncodeURL(&EventsOptions{
			Start:  now,
			End:    now,
			Filter: "query",
			Limit:  100})

		expected := fmt.Sprintf("https://pull.logentries.com/key/hosts/host/log/?end=%d&filter=query&limit=100&start=%d", now, now)

		require.Equal(t, expected, actual)
	})

	n.Meow()
}

func TestNewEvents(t *testing.T) {
	n := neko.Start(t)

	n.It("unmarshals error response", func() {
		body := []byte(`
{
	"response":"error",
	"reason":"Account 49cec5a8-01e1-4461-abbf-c20c7e7faad1 not found"
}
		`)

		_, err := NewEvents(body)

		require.Error(t, err)
	})

	n.It("unmarshals successful response", func() {
		body := []byte(`
{"@timestamp":"2015-03-25T21:09:04.335940327Z","@type":"log","@version":"1","message":"awesome"}
{"@timestamp":"2015-03-25T21:11:22.648632566Z","@type":"log","@version":"1","message":"totally"}
{"@timestamp":"2015-03-25T21:12:20.888999096Z","@type":"log","@version":"1","message":"radical"}`)

		events, err := NewEvents(body)

		require.NoError(t, err)
		require.Equal(t, 3, len(events))

		message := events[0]

		msg, ok := message.GetString("message")
		require.True(t, ok)

		require.Equal(t, "awesome", msg)

		timestamp, _ := time.Parse(time.RFC3339Nano, "2015-03-25T21:09:04.335940327Z")

		require.Equal(t, timestamp, message.GetTimestamp().Time())
		require.Equal(t, "log", message.StringType())
		require.Equal(t, 1, message.GetVersion())
	})

	n.It("creates new cypress messages if can't unmarshall response", func() {
		body := []byte("awesome\ntotally\nradical")

		events, err := NewEvents(body)

		require.NoError(t, err)
		require.Equal(t, 3, len(events))

		message := events[0]

		msg, ok := message.GetString("message")
		require.True(t, ok)

		require.Equal(t, "awesome", msg)
		require.Equal(t, "log", message.StringType())
		require.Equal(t, 1, message.GetVersion())
	})

	n.Meow()
}

func TestBufferEvents(t *testing.T) {
	n := neko.Start(t)

	defaults := &EventsOptions{}

	n.It("adds event to events buffer", func() {
		lr, _ := NewLogentriesRecv("key", "host", "log", defaults, 100)

		var events []*cypress.Message
		expected := cypress.Log()
		events = append(events, expected)

		err := lr.BufferEvents(events)
		require.NoError(t, err)

		actual := <-lr.EventBuffer

		require.Equal(t, expected, actual)
	})

	n.It("sets start to be the timestamp from the added event", func() {
		lr, _ := NewLogentriesRecv("key", "host", "log", defaults, 100)

		var events []*cypress.Message
		expected := cypress.Log()
		events = append(events, expected)

		err := lr.BufferEvents(events)
		require.NoError(t, err)

		require.Equal(t, milliseconds(expected.GetTimestamp().Time()), lr.Options.Start)
	})

	n.It("does not wait on full buffer", func() {
		lr, _ := NewLogentriesRecv("key", "host", "log", defaults, 1)

		var events []*cypress.Message
		expected := cypress.Log()
		extra := cypress.Log()
		events = append(events, expected)
		events = append(events, extra)

		err := lr.BufferEvents(events)
		require.NoError(t, err)

		require.Equal(t, milliseconds(expected.GetTimestamp().Time()), lr.Options.Start)
	})

	n.Meow()
}

func TestLogentriesOnline(t *testing.T) {
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

	key := os.Getenv(cAccountKey)
	if key == "" {
		t.Skipf("%s is not set.", cAccountKey)
	}

	host := os.Getenv(cHost)
	if host == "" {
		t.Skipf("%s is not set.", cHost)
	}

	log := os.Getenv(cLog)
	if log == "" {
		t.Skipf("%s is not set.", cLog)
	}

	// Send message to logentries

	l := NewLogger(endpoint, ssl == "true", token)
	go l.Run()

	expected := tcplog.NewMessage(t)
	l.Read(expected)

	time.Sleep(10 * time.Second)

	// Read back message from logentries

	timestamp, _ := json.Marshal(expected.Timestamp)

	options := &EventsOptions{Filter: fmt.Sprintf("/%s", timestamp)}
	lr, err := NewLogentriesRecv(key, host, log, options, 100)
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

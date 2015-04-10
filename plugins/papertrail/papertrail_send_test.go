package papertrail

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/vektra/cypress"
	"github.com/vektra/cypress/plugins/lib/tcplog"
)

const cTimeFormat = time.RFC3339Nano

func TestPapertrailFormat(t *testing.T) {
	l := NewLogger("", false)

	message := cypress.Log()
	message.Add("message", "the message")
	message.AddString("string_key", "I'm a string!")
	message.AddInt("int_key", 12)
	message.AddBytes("bytes_key", []byte("I'm bytes!"))
	message.AddInterval("interval_key", 2, 1)

	actual, err := l.Format(message)
	if err != nil {
		t.Errorf("Error formatting: %s", err)
	}

	timestamp := message.GetTimestamp().Time().Format(cTimeFormat)

	expected := fmt.Sprintf("{\"@timestamp\":\"%s\",\"@type\":\"log\",\"@version\":\"1\",\"bytes_key\":{\"bytes\":\"SSdtIGJ5dGVzIQ==\"},\"int_key\":12,\"interval_key\":{\"nanoseconds\":1,\"seconds\":2},\"message\":\"the message\",\"string_key\":\"I'm a string!\"}\n", timestamp)

	assert.Equal(t, expected, string(actual))
}

func TestPapertrailRunWithTestServer(t *testing.T) {
	s := tcplog.NewTcpServer()
	go s.Run("127.0.0.1")

	l := NewLogger(<-s.Address, false)

	message := tcplog.NewMessage(t)
	l.Read(message)

	select {
	case m := <-s.Messages:
		expected, err := l.Format(message)
		if err != nil {
			t.Errorf("Error formatting: %s", err)
		}

		assert.Equal(t, string(expected), string(m))

	case <-time.After(5 * time.Second):
		t.Errorf("Test server did not get message in time.")
	}
}

package loggly

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/vektra/cypress"
)

const cTimeFormat = time.RFC3339Nano
const cEndpoint = "TEST_LOGGLY_URL"
const cSSL = "TEST_LOGGLY_SSL"
const cToken = "TEST_LOGGLY_TOKEN"
const cPEN = "TEST_LOGGLY_PEN"

func TestLogglyFormat(t *testing.T) {
	l := NewLogger("token")

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

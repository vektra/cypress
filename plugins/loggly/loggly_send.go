package loggly

import (
	"encoding/json"

	client "github.com/segmentio/go-loggly"
	"github.com/vektra/cypress"
)

const cNewline = "\n"

type Logger struct {
	*client.Client
}

func NewLogger(token string) *Logger {
	return &Logger{client.New(token)}
}

func (l *Logger) Format(m *cypress.Message) ([]byte, error) {
	bytes, err := json.Marshal(m)
	if err != nil {
		return nil, err
	}

	bytes = append(bytes, []byte(cNewline)...)

	return bytes, nil
}

func (l *Logger) Receive(message *cypress.Message) error {
	bytes, err := l.Format(message)
	if err != nil {
		return err
	}

	_, err = l.Write(bytes)
	if err != nil {
		return err
	}

	return nil
}

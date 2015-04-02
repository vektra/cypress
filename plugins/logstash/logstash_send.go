package logstash

import (
	"encoding/json"

	"github.com/vektra/cypress"
	"github.com/vektra/cypress/plugins/lib/tcplog"
)

const cNewline = "\n"

type LogstashFormatter struct{}

func NewLogger(address string, ssl bool) *tcplog.Logger {
	return tcplog.NewLogger(address, ssl, &LogstashFormatter{})
}

func (lf *LogstashFormatter) Format(m *cypress.Message) ([]byte, error) {
	bytes, err := json.Marshal(m)
	if err != nil {
		return nil, err
	}

	bytes = append(bytes, []byte(cNewline)...)

	return bytes, nil
}

package logstash

import (
	"os"

	"github.com/vektra/cypress"
	"github.com/vektra/cypress/cli/commands"
)

type Send struct {
	Host string `short:"H" long:"host" description:"Logstash host <host>:<port>"`
	Ssl  bool   `short:"S" long:"tls" default:"false" description:"Use TLS"`
}

func (p *Send) Execute(args []string) error {
	logstash := NewLogger(p.Host, p.Ssl)

	dec, err := cypress.NewStreamDecoder(os.Stdin)
	if err != nil {
		return err
	}

	return cypress.Glue(dec, logstash)
}

type Plugin struct {
	Host string
	Ssl  bool
}

func (p *Plugin) Receiver() (cypress.Receiver, error) {
	return NewLogger(p.Host, p.Ssl), nil
}

func init() {
	short := "Send messages to Logstash"
	long := "Given a stream on stdin, the logstash command will read those messages in and send them to Logstash via TCP."

	commands.Add("logstash:send", short, long, &Send{})

	cypress.AddPlugin("logstash", func() cypress.Plugin { return &Send{} })
}

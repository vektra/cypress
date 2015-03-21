package cli

import (
	"fmt"
	"os"

	"github.com/vektra/addons/logstash"
	"github.com/vektra/cypress"
)

type LogstashCommand struct {
	Host string `short:"H" long:"host" description:"Logstash host <host>:<port>"`
	Port string `short:"P" long:"port" description:"Logstash port <host>:<port>"`
	Ssl  bool   `short:"S" long:"tls" default:"true" description:"Use TLS"`
}

func (p *LogstashCommand) Execute(args []string) error {
	address := fmt.Sprintf("%s:%s", p.Host, p.Port)

	logstash := logstash.NewLogger(address, p.Ssl)

	dec, err := cypress.NewStreamDecoder(os.Stdin)
	if err != nil {
		return err
	}

	logstash.Run()

	return cypress.Glue(dec, logstash)
}

func init() {
	short := "Send messages to Logstash"
	long := "Given a stream on stdin, the logstash command will read those messages in and send them to Logstash via TCP."

	addCommand("logstash", short, long, &LogstashCommand{})
}

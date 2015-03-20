package cli

import (
	"fmt"
	"os"

	"github.com/vektra/addons/loggly"
	"github.com/vektra/cypress"
)

type LogglyCommand struct {
	Host  string `short:"H" long:"host" default:"logs-01.loggly.com" description:"Loggly host <host>:<port>"`
	Port  string `short:"P" long:"port" default:"6514" description:"Loggly port <host>:<port>"`
	Ssl   bool   `short:"S" long:"tls" default:"true" description:"Use TLS"`
	Token string `short:"T" long:"token" description:"Loggly customer token"`
	PEN   string `long:"pen" default:"41058" description:"Loggly private enterprise number"`
}

func (p *LogglyCommand) Execute(args []string) error {
	address := fmt.Sprintf("%s:%s", p.Host, p.Port)

	loggly := loggly.NewLogger(address, p.Ssl, p.Token, p.PEN)

	dec, err := cypress.NewStreamDecoder(os.Stdin)
	if err != nil {
		return err
	}

	loggly.Run()

	return cypress.Glue(dec, loggly)
}

func init() {
	short := "Send messages to Loggly"
	long := "Given a stream on stdin, the loggly command will read those messages in and send them to Loggly via TCP."

	addCommand("loggly", short, long, &LogglyCommand{})
}

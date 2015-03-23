package cli

import (
	"fmt"
	"os"

	"github.com/vektra/addons/logentries"
	"github.com/vektra/cypress"
)

type LogentriesCommand struct {
	Host  string `short:"H" long:"host" default:"data.logentries.com" description:"Logentries host <host>:<port>"`
	Port  string `short:"P" long:"port" default:"2000" description:"Logentries port <host>:<port>"`
	Ssl   bool   `short:"S" long:"tls" default:"true" description:"Use TLS"`
	Token string `short:"T" long:"token" description:"Logentries token that uniquely identifies the destination log"`
}

func (p *LogentriesCommand) Execute(args []string) error {
	address := fmt.Sprintf("%s:%s", p.Host, p.Port)

	logentries := logentries.NewLogger(address, p.Ssl, p.Token)

	dec, err := cypress.NewStreamDecoder(os.Stdin)
	if err != nil {
		return err
	}

	logentries.Run()

	return cypress.Glue(dec, logentries)
}

func init() {
	short := "Send messages to Logentries"
	long := "Given a stream on stdin, the logentries command will read those messages in and send them to Logentries via TCP."

	addCommand("logentries", short, long, &LogentriesCommand{})
}

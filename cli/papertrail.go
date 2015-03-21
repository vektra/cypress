package cli

import (
	"fmt"
	"os"

	"github.com/vektra/addons/papertrail"
	"github.com/vektra/cypress"
)

type PapertrailCommand struct {
	Host string `short:"H" long:"host" description:"Papertrail host <host>:<port>"`
	Port string `short:"P" long:"port" description:"Papertrail port <host>:<port>"`
	Ssl  bool   `short:"S" long:"tls" default:"true" description:"Use TLS"`
}

func (p *PapertrailCommand) Execute(args []string) error {
	address := fmt.Sprintf("%s:%s", p.Host, p.Port)

	papertrail := papertrail.NewLogger(address, p.Ssl)

	dec, err := cypress.NewStreamDecoder(os.Stdin)
	if err != nil {
		return err
	}

	papertrail.Run()

	return cypress.Glue(dec, papertrail)
}

func init() {
	short := "Send messages to Papertrail"
	long := "Given a stream on stdin, the papertrail command will read those messages in and send them to Papertrail via TCP."

	addCommand("papertrail", short, long, &PapertrailCommand{})
}

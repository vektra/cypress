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

type PapertrailGenerateCommand struct {
	Token      string `short:"T" long:"token" description:"Papertrail token"`
	Q          string `short:"q" long:"query" description:"Query string"`
	GroupId    string `long:"groupid"`
	SystemId   string `long:"systemid"`
	MinId      string `long:"minid"`
	MaxId      string `long:"maxid"`
	MinTime    string `long:"mintime"`
	MaxTime    string `long:"maxtime"`
	BufferSize int    `long:"buffersize" default:"100"`
}

func (g *PapertrailGenerateCommand) Execute(args []string) error {
	options := &papertrail.EventsOptions{
		Q:        g.Q,
		GroupId:  g.GroupId,
		SystemId: g.SystemId,
		MinId:    g.MinId,
		MaxId:    g.MaxId,
		MinTime:  g.MinTime,
		MaxTime:  g.MaxTime,
	}

	generator := papertrail.NewAPIClient(g.Token, options, g.BufferSize)

	receiver := os.Stdout

	return cypress.Glue(generator, receiver)
}

func init() {
	short := "Send messages to Papertrail"
	long := "Given a stream on stdin, the papertrail command will read those messages in and send them to Papertrail via TCP."

	addCommand("papertrail", short, long, &PapertrailCommand{})

	short = "Get messages from Papertrail"
	long = ""

	addCommand("papertrail:generate", short, long, &PapertrailGenerateCommand{})
}

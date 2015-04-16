package papertrail

import (
	"os"

	"github.com/vektra/cypress"
	"github.com/vektra/cypress/cli/commands"
)

type Send struct {
	Host string `short:"H" long:"host" description:"Papertrail host <host>:<port>"`
	Ssl  bool   `short:"S" long:"tls" default:"false" description:"Use TLS"`
}

func (p *Send) Execute(args []string) error {
	papertrail := NewLogger(p.Host, p.Ssl)

	dec, err := cypress.NewStreamDecoder(os.Stdin)
	if err != nil {
		return err
	}

	return cypress.Glue(dec, papertrail)
}

type Recv struct {
	Token      string `short:"T" long:"token" description:"Papertrail token"`
	Q          string `short:"q" long:"query" description:"Papertrail search query"`
	GroupId    string `long:"groupid" description:"Limit results to specific Papertrail group"`
	SystemId   string `long:"systemid" description:"Limit results to specific Papertrail system"`
	MinId      string `long:"minid" description:"The oldest Papertrail message ID to examine"`
	MaxId      string `long:"maxid" description:"The newest Papertrail message ID to examine"`
	MinTime    string `long:"mintime" description:"The oldest Papertrail timestamp to examine"`
	MaxTime    string `long:"maxtime" description:"The newest Papertrail timestamp to examine"`
	Tail       bool   `long:"tail" default:"false" description:"Set to true to search newest to oldest"`
	BufferSize int    `long:"buffersize" default:"100"`
}

func (g *Recv) Execute(args []string) error {
	options := EventsOptions{
		Q:        g.Q,
		GroupId:  g.GroupId,
		SystemId: g.SystemId,
		MinId:    g.MinId,
		MaxId:    g.MaxId,
		MinTime:  g.MinTime,
		MaxTime:  g.MaxTime,
		Tail:     g.Tail,
	}

	generator := NewPapertrailRecv(g.Token, &options, g.BufferSize)

	receiver := cypress.NewStreamEncoder(os.Stdout)

	return cypress.Glue(generator, receiver)
}

type Plugin struct {
	Host string
	Ssl  bool

	Token    string
	Q        string
	GroupId  string
	SystemId string
	MinId    string
	MaxId    string
	MinTime  string
	MaxTime  string

	Tail       bool
	BufferSize int
}

func (p *Plugin) Receiver() (cypress.Receiver, error) {
	return NewLogger(p.Host, p.Ssl), nil
}

func (p *Plugin) Generator() (cypress.Generator, error) {
	options := EventsOptions{
		Q:        p.Q,
		GroupId:  p.GroupId,
		SystemId: p.SystemId,
		MinId:    p.MinId,
		MaxId:    p.MaxId,
		MinTime:  p.MinTime,
		MaxTime:  p.MaxTime,
		Tail:     p.Tail,
	}

	return NewPapertrailRecv(p.Token, &options, p.BufferSize), nil
}

func init() {
	short := "Send messages to Papertrail"
	long := "Given a stream on stdin, the papertrail command will read those messages in and send them to Papertrail via TCP."

	commands.Add("papertrail:send", short, long, &Send{})

	short = "Get messages from Papertrail"
	long = ""

	commands.Add("papertrail:recv", short, long, &Recv{})

	cypress.AddPlugin("papertrail", func() cypress.Plugin {
		return &Plugin{BufferSize: 100}
	})
}

package logentries

import (
	"os"

	"github.com/vektra/cypress"
	"github.com/vektra/cypress/cli/commands"
)

type Send struct {
	Host  string `short:"H" long:"host" default:"data.logentries.com:2000" description:"Logentries host <host>:<port>"`
	Ssl   bool   `short:"S" long:"tls" description:"Use TLS"`
	Token string `short:"T" long:"token" description:"Logentries token that uniquely identifies the destination log"`
}

func (p *Send) Execute(args []string) error {
	logentries := NewLogger(p.Host, p.Ssl, p.Token)

	dec, err := cypress.NewStreamDecoder(os.Stdin)
	if err != nil {
		return err
	}

	return cypress.Glue(dec, logentries)
}

type Recv struct {
	Key  string `short:"k" long:"key" description:"Logentries token"`
	Host string `short:"h" long:"host" description:"Logentries host name"`
	Log  string `short:"l" long:"log" description:"Logentries log name"`

	Start  int    `long:"start" description:"Starting point, timestamp in milliseconds since Epoch 1970-01-01 00:00:00 +0000 (UTC). Alternatively, a negative value represents milliseconds before the current time."`
	End    int    `long:"end" description:"Ending point, timestamp in milliseconds since Epoch 1970-01-01 00:00:00 +0000 (UTC). Alternatively, a negative value represents milliseconds before the current time."`
	Filter string `long:"filter" description:"Filtering pattern. It is a keyword or a regular expression prepended with slash (/)."`
	Limit  int    `long:"limit" default:"100" description:"Maximal number of events downloaded."`

	BufferSize int `long:"buffersize" default:"100"`
}

func (g *Recv) Execute(args []string) error {
	options := EventsOptions{
		Start:  g.Start,
		End:    g.End,
		Filter: g.Filter,
		Limit:  g.Limit,
	}

	generator, err := NewLogentriesRecv(g.Key, g.Host, g.Log, &options, g.BufferSize)
	if err != nil {
		return err
	}

	receiver := cypress.NewStreamEncoder(os.Stdout)

	return cypress.Glue(generator, receiver)
}

type Plugin struct {
	Host  string
	Ssl   bool
	Token string

	Key string
	Log string

	Start      int
	End        int
	Filter     string
	Limit      int
	BufferSize int
}

func (p *Plugin) Generator() (cypress.Generator, error) {
	options := EventsOptions{
		Start:  p.Start,
		End:    p.End,
		Filter: p.Filter,
		Limit:  p.Limit,
	}

	return NewLogentriesRecv(p.Key, p.Host, p.Log, &options, p.BufferSize)
}

func (p *Plugin) Receiver() (cypress.Receiver, error) {
	return NewLogger(p.Host, p.Ssl, p.Token), nil
}

func init() {
	short := "Send messages to Logentries"
	long := "Given a stream on stdin, the logentries command will read those messages in and send them to Logentries via TCP."

	commands.Add("logentries:send", short, long, &Send{})

	short = "Get messages from Logentries"
	long = ""

	commands.Add("logentries:recv", short, long, &Recv{})

	cypress.AddPlugin("logentries", func() cypress.Plugin {
		return &Plugin{
			Host:       "data.logentries.com:2000",
			Limit:      100,
			BufferSize: 100,
		}
	})
}

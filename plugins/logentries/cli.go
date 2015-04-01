package logentries

import (
	"fmt"
	"os"

	"github.com/vektra/cypress"
	"github.com/vektra/cypress/cli/commands"
)

type Send struct {
	Host  string `short:"H" long:"host" default:"data.logentries.com" description:"Logentries host <host>:<port>"`
	Port  string `short:"P" long:"port" default:"2000" description:"Logentries port <host>:<port>"`
	Ssl   bool   `short:"S" long:"tls" default:"true" description:"Use TLS"`
	Token string `short:"T" long:"token" description:"Logentries token that uniquely identifies the destination log"`
}

func (p *Send) Execute(args []string) error {
	address := fmt.Sprintf("%s:%s", p.Host, p.Port)

	logentries := NewLogger(address, p.Ssl, p.Token)

	dec, err := cypress.NewStreamDecoder(os.Stdin)
	if err != nil {
		return err
	}

	logentries.Run()

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

func init() {
	short := "Send messages to Logentries"
	long := "Given a stream on stdin, the logentries command will read those messages in and send them to Logentries via TCP."

	commands.Add("send", short, long, &Send{})

	short = "Get messages from Logentries"
	long = ""

	commands.Add("recv", short, long, &Recv{})
}

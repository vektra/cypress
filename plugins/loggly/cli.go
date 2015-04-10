package loggly

import (
	"os"

	"github.com/vektra/cypress"
	"github.com/vektra/cypress/cli/commands"
)

type Send struct {
	Token string `short:"T" long:"token" description:"Loggly token that uniquely identifies the destination log"`
}

func (p *Send) Execute(args []string) error {
	loggly := NewLogger(p.Token)

	dec, err := cypress.NewStreamDecoder(os.Stdin)
	if err != nil {
		return err
	}

	return cypress.Glue(dec, loggly)
}

type Recv struct {
	Account  string `short:"a" long:"account" description:"Loggly account name"`
	Username string `short:"u" long:"username" description:"Loggly username"`
	Password string `short:"p" long:"password" description:"Loggly password"`

	Q     string `short:"q" long:"query" description:"Loggly search query"`
	From  string `long:"from" default:"-24h" description:"Start time for the search."`
	Until string `long:"until" default:"now" description:"End time for the search."`
	Order string `long:"order" default:"desc" description:"Direction of results returned, either asc or desc."`
	Size  uint   `long:"size" default:"100" description:"Number of rows returned by search."`

	BufferSize int `long:"buffersize" default:"100"`
}

func (g *Recv) Execute(args []string) error {
	rsid := RSIDOptions{
		Q:     g.Q,
		From:  g.From,
		Until: g.Until,
		Order: g.Order,
		Size:  g.Size,
	}

	options := EventsOptions{}

	generator, err := NewLogglyRecv(g.Account, g.Username, g.Password, &rsid, &options, g.BufferSize)
	if err != nil {
		return err
	}

	receiver := cypress.NewStreamEncoder(os.Stdout)

	return cypress.Glue(generator, receiver)
}

func init() {
	short := "Send messages to Loggly"
	long := "Given a stream on stdin, the loggly command will read those messages in and send them to Loggly via TCP."

	commands.Add("loggly:send", short, long, &Send{})

	short = "Get messages from Loggly"
	long = ""

	commands.Add("loggly:recv", short, long, &Recv{})
}

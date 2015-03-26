package statsd

import (
	"os"

	"github.com/vektra/cypress"
	"github.com/vektra/cypress/cli/commands"
)

type CLI struct {
	Listen string `short:"l" long:"listen" default:":8125" description:"UDP host:port to listen on"`
}

func (s *CLI) Execute(args []string) error {
	enc := cypress.NewStreamEncoder(os.Stdout)

	err := enc.Init(cypress.SNAPPY)
	if err != nil {
		return err
	}

	ep, err := NewStatsdEndpoint(enc, s.Listen)
	if err != nil {
		return err
	}

	return ep.Run()
}

func init() {
	commands.Add("statsd", "listen on statsd and generate metrics", "", &CLI{})
}

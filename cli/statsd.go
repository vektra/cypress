package cli

import (
	"os"

	"github.com/vektra/cypress"
	"github.com/vektra/cypress/plugin"
)

type Statsd struct {
	Listen string `short:"l" long:"listen" default:":8125" description:"UDP host:port to listen on"`
}

func (s *Statsd) Execute(args []string) error {
	enc := cypress.NewStreamEncoder(os.Stdout)

	err := enc.Init(cypress.SNAPPY)
	if err != nil {
		return err
	}

	ep, err := plugin.NewStatsdEndpoint(enc, s.Listen)
	if err != nil {
		return err
	}

	return ep.Run()
}

func init() {
	addCommand("statsd", "listen on statsd and generate metrics", "", &Statsd{})
}

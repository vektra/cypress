package cli

import (
	"os"

	"github.com/vektra/cypress"
)

type Send struct {
	Addr   string `short:"a" long:"addr" description:"Who to send the stream to"`
	Window int    `short:"w" long:"window" description:"Window size to use when transmitting"`
}

func (s *Send) Execute(args []string) error {
	window := s.Window
	if window == 0 {
		window = cypress.MinimumWindow
	}

	tcp, err := cypress.NewTCPSend(s.Addr)
	if err != nil {
		return err
	}

	r, err := cypress.NewStreamDecoder(os.Stdin)
	if err != nil {
		return err
	}

	return cypress.Glue(r, tcp)
}

func init() {
	addCommand("send", "send a stream to a remote place", "", &Send{})
}

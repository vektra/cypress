package cli

import (
	"os"
	"strings"

	"github.com/vektra/cypress"
	"github.com/vektra/cypress/plugins/tcp"
)

type Send struct {
	Addr   string `short:"a" long:"addr" description:"Who to send the stream to"`
	Window int    `short:"w" long:"window" description:"Window size to use when transmitting"`
	Buffer int    `short:"b" long:"buffer" description:"How big of an internal buffer to use"`
}

func (s *Send) Execute(args []string) error {
	window := s.Window
	if window == 0 {
		window = cypress.MinimumWindow
	}

	buffer := s.Buffer
	if buffer == 0 {
		buffer = tcp.DefaultTCPBuffer
	}

	addrs := strings.Split(s.Addr, ",")

	tcp, err := tcp.NewTCPSend(addrs, window, buffer)
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

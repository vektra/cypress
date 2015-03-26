package tcp

import (
	"os"
	"strings"

	"github.com/vektra/cypress"
	"github.com/vektra/cypress/cli/commands"
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
		buffer = DefaultTCPBuffer
	}

	addrs := strings.Split(s.Addr, ",")

	tcp, err := NewTCPSend(addrs, window, buffer)
	if err != nil {
		return err
	}

	r, err := cypress.NewStreamDecoder(os.Stdin)
	if err != nil {
		return err
	}

	return cypress.Glue(r, tcp)
}

type Recv struct {
	Listen string `short:"l" long:"listen" description:"host:port to listen on"`
}

func (r *Recv) Execute(args []string) error {
	tcp, err := NewTCPRecvGenerator(r.Listen)
	if err != nil {
		return err
	}

	return cypress.Glue(tcp, cypress.NewStreamEncoder(os.Stdout))
}

func init() {
	commands.Add("send", "send a stream to a remote host(s)", "", &Send{})
	commands.Add("recv", "accept streams over the network", "", &Recv{})
}

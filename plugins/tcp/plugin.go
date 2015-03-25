package tcp

import (
	"net"

	"github.com/vektra/cypress"
)

type TCPPlugin struct {
	Address  string
	Listener net.Listener
}

func (r *TCPPlugin) Receiver() (cypress.Receiver, error) {
	return NewTCPSend(r.Address, 0, DefaultTCPBuffer)
}

func (r *TCPPlugin) Generator() (cypress.Generator, error) {
	return NewTCPRecvGenerator(r.Address)
}

func init() {
	cypress.AddPlugin("TCP", func() cypress.Plugin {
		return &TCPPlugin{}
	})
}

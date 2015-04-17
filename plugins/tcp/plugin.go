package tcp

import "github.com/vektra/cypress"

type TCPPlugin struct {
	Address string `description:"host:port to listen (input) or send to (output)"`
}

func (t *TCPPlugin) Description() string {
	return `Send or receive message streams over TCP.`
}

func (r *TCPPlugin) Receiver() (cypress.Receiver, error) {
	return NewTCPSend([]string{r.Address}, 0, DefaultTCPBuffer)
}

func (r *TCPPlugin) Generator() (cypress.Generator, error) {
	return NewTCPRecvGenerator(r.Address)
}

func init() {
	cypress.AddPlugin("TCP", func() cypress.Plugin {
		return &TCPPlugin{}
	})
}

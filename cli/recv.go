package cli

import (
	"net"
	"os"
	"sync"

	"github.com/vektra/cypress"
)

type Recv struct {
	Listen string `short:"l" long:"listen" description:"host:port to listen on"`

	lock sync.Mutex

	enc *cypress.StreamEncoder
}

func (r *Recv) Execute(args []string) error {
	l, err := net.Listen("tcp", r.Listen)
	if err != nil {
		return err
	}

	r.enc = cypress.NewStreamEncoder(os.Stdout)

	for {
		c, err := l.Accept()
		if err != nil {
			return err
		}

		go r.handle(c)
	}

	return nil
}

func (r *Recv) handle(c net.Conn) error {
	recv, err := cypress.NewRecv(c)
	if err != nil {
		return err
	}

	return cypress.Glue(recv, r)
}

func (r *Recv) Receive(m *cypress.Message) error {
	r.lock.Lock()
	defer r.lock.Unlock()

	return r.enc.Receive(m)
}

func init() {
	addCommand("recv", "accept streams over the network", "", &Recv{})
}

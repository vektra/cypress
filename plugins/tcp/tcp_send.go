package tcp

import (
	"errors"
	"math/rand"
	"net"

	"github.com/vektra/cypress"
)

type TCPSend struct {
	*cypress.ReliableSend

	hosts  []string
	window int
	c      net.Conn
}

const DefaultTCPBuffer = 128

func NewTCPSend(hosts []string, window, buffer int) (*TCPSend, error) {
	tcp := &TCPSend{
		hosts:  hosts,
		window: window,
	}

	tcp.ReliableSend = cypress.NewReliableSend(tcp, buffer)

	err := tcp.Start()
	if err != nil {
		return nil, err
	}

	return tcp, nil
}

var ErrNoAvailableHosts = errors.New("no available hosts")

func shuffle(a []string) {
	for i := range a {
		j := rand.Intn(i + 1)
		a[i], a[j] = a[j], a[i]
	}
}

func (t *TCPSend) Connect() (*cypress.Send, error) {
	shuffle(t.hosts)

	for _, host := range t.hosts {
		c, err := net.Dial("tcp", host)
		if err != nil {
			continue
		}

		s := cypress.NewSend(c, t.window)
		err = s.SendHandshake()
		if err != nil {
			c.Close()
			continue
		}

		t.c = c

		return s, nil
	}

	return nil, ErrNoAvailableHosts
}

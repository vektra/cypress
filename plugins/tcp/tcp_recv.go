package tcp

import (
	"io"
	"net"

	"github.com/vektra/cypress"
)

type TCPRecv struct {
	Addr    string
	Handler cypress.GeneratorHandler

	l net.Listener
}

func NewTCPRecv(host string, h cypress.GeneratorHandler) (*TCPRecv, error) {
	return &TCPRecv{Addr: host, Handler: h}, nil
}

func (t *TCPRecv) Close() error {
	return t.l.Close()
}

func (t *TCPRecv) Listen() error {
	l, err := net.Listen("tcp", t.Addr)
	if err != nil {
		return err
	}

	t.l = l

	return nil
}

func (t *TCPRecv) Accept() error {
	for {
		c, err := t.l.Accept()
		if err != nil {
			return err
		}

		go t.handle(c)
	}

	return nil
}

func (t *TCPRecv) ListenAndAccept() error {
	err := t.Listen()
	if err != nil {
		return err
	}

	return t.Accept()
}

func (t *TCPRecv) handle(c net.Conn) {
	recv, err := cypress.NewRecv(c)
	if err != nil {
		return
	}

	t.Handler.HandleGenerator(recv)
}

type TCPRecvGenerator struct {
	*TCPRecv

	buf chan *cypress.Message
}

func NewTCPRecvGenerator(host string) (*TCPRecvGenerator, error) {
	g := &TCPRecvGenerator{
		buf: make(chan *cypress.Message, 10),
	}

	tcp, err := NewTCPRecv(host, g)
	if err != nil {
		return nil, err
	}

	g.TCPRecv = tcp

	err = g.Listen()
	if err != nil {
		return nil, err
	}

	go g.Accept()

	return g, nil
}

func (t *TCPRecvGenerator) Run(r cypress.Receiver) error {
	for {
		m, ok := <-t.buf
		if !ok {
			return io.EOF
		}

		err := r.Receive(m)
		if err != nil {
			return err
		}
	}

	return nil
}

func (t *TCPRecvGenerator) Generate() (*cypress.Message, error) {
	m, ok := <-t.buf
	if !ok {
		return nil, io.EOF
	}

	return m, nil
}

func (t *TCPRecvGenerator) Close() error {
	t.TCPRecv.Close()
	close(t.buf)

	return nil
}

func (t *TCPRecvGenerator) HandleGenerator(g cypress.Generator) {
	for {
		m, err := g.Generate()
		if err != nil {
			return
		}

		t.buf <- m
	}
}

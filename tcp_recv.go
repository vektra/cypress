package cypress

import "net"

type TCPRecv struct {
	Addr    string
	Handler GeneratorHandler

	l net.Listener
}

func NewTCPRecv(host string, h GeneratorHandler) (*TCPRecv, error) {
	return &TCPRecv{Addr: host, Handler: h}, nil
}

func (t *TCPRecv) ListenAndAccept() error {
	l, err := net.Listen("tcp", t.Addr)
	if err != nil {
		return err
	}

	for {
		c, err := l.Accept()
		if err != nil {
			return err
		}

		go t.handle(c)
	}

	return nil
}

func (t *TCPRecv) handle(c net.Conn) {
	recv, err := NewRecv(c)
	if err != nil {
		return
	}

	t.Handler.HandleGenerator(recv)
}

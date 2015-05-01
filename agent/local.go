package agent

import (
	"container/list"
	"fmt"
	"io"
	"net"
	"os"
	"path"
	"strings"
	"sync"

	"github.com/vektra/cypress"
)

const cPlus = '+'

type ManyReceiver struct {
	recievers []cypress.Receiver
}

func (mr *ManyReceiver) Receive(m *cypress.Message) error {
	for _, r := range mr.recievers {
		err := r.Receive(m)
		if err != nil {
			return err
		}
	}

	return nil
}

func (mr *ManyReceiver) Close() error {
	var err error

	for _, r := range mr.recievers {
		if e := r.Close(); e != nil {
			err = e
		}
	}

	return err
}

func ManyReceivers(r ...cypress.Receiver) *ManyReceiver {
	return &ManyReceiver{r}
}

func LocalCollector(r cypress.Receiver) Source {
	return newServer(cypress.LogPath(), r)
}

func newServer(path string, r cypress.Receiver) *server {
	return &server{
		path:  path,
		conns: list.New(),
		recv:  r,
	}
}

type server struct {
	mu sync.Mutex

	path string

	server net.Listener
	conns  *list.List
	recv   cypress.Receiver
}

type Source interface {
	Start() error
	Close()
}

func (s *server) removeConn(e *list.Element) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.conns.Remove(e)
}

func (s *server) trackConn(c net.Conn) *list.Element {
	return s.conns.PushBack(c)
}

func (s *server) closeConns() {
	s.mu.Lock()
	defer s.mu.Unlock()

	e := s.conns.Front()

	for e != nil {
		e.Value.(net.Conn).Close()
		e = e.Next()
	}
}

func (s *server) serve(c net.Conn, e *list.Element) {
	dec := cypress.NewDecoder(c)

	for {
		m, err := dec.Decode()
		if err != nil {
			if err == io.EOF {
				c.Close()
				s.removeConn(e)
				return
			}

			fmt.Printf("Error reading message: %s\n", err)
		}

		s.recv.Receive(m)
	}
}

func (s *server) Close() {
	s.server.Close()
}

func (s *server) Start() error {
	os.RemoveAll(s.path)

	os.MkdirAll(path.Dir(s.path), 0755)

	l, err := net.Listen("unix", s.path)

	if err != nil {
		return err
	}

	os.Chmod(s.path, 0777)

	s.server = l

	for {
		cl, err := l.Accept()

		if err != nil {
			if err == io.EOF {
				s.closeConns()
				return nil
			}

			if strings.Index("closed network connection", err.Error()) != -1 {
				return nil
			}
		} else {
			e := s.trackConn(cl)
			go s.serve(cl, e)
		}
	}

	return nil
}

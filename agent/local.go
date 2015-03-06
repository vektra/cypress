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
const cExclamation = '!'

type ManyReciever struct {
	recievers []cypress.Reciever
}

func (mr *ManyReciever) Read(m *cypress.Message) error {
	for _, r := range mr.recievers {
		err := r.Read(m)
		if err != nil {
			return err
		}
	}

	return nil
}

func ManyRecievers(r ...cypress.Reciever) *ManyReciever {
	return &ManyReciever{r}
}

func LocalCollector(r cypress.Reciever) Source {
	return newServer(cypress.LogPath(), r)
}

func newServer(path string, r cypress.Reciever) *server {
	return &server{
		path:  path,
		conns: list.New(),
		recv:  r,
		taps:  ManyRecievers(),
	}
}

type server struct {
	mu sync.Mutex

	path string

	server net.Listener
	conns  *list.List
	recv   cypress.Reciever
	taps   *ManyReciever
}

type tap struct {
	c   net.Conn
	idx int
}

type Source interface {
	Start() error
	Close()
}

func (t *tap) Read(m *cypress.Message) error {
	_, err := m.WriteWire(t.c)
	return err
}

func (s *server) addTap(c net.Conn) *tap {
	s.mu.Lock()
	defer s.mu.Unlock()

	to := &tap{c, len(s.taps.recievers)}
	s.taps.recievers = append(s.taps.recievers, to)
	return to
}

func (s *server) removeTap(t *tap) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.taps.recievers = append(s.taps.recievers[:t.idx],
		s.taps.recievers[t.idx+1:]...)
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
	var buf [1]byte
	var tapped *tap

	for {
		_, err := c.Read(buf[:])

		if err != nil {
			if err == io.EOF {
				if tapped != nil {
					s.removeTap(tapped)
				}

				c.Close()
				s.removeConn(e)
				return
			}

			fmt.Printf("Error reading message: %s\n", err)
		}

		switch buf[0] {
		case cPlus:
			m := &cypress.Message{}
			_, err := m.ReadWire(c)
			if err != nil {
				if err == io.EOF {
					if tapped != nil {
						s.removeTap(tapped)
					}

					s.removeConn(e)
					c.Close()
					return
				}

				fmt.Printf("Error reading message: %s\n", err)
			} else {
				s.taps.Read(m)
				s.recv.Read(m)
			}
		case cExclamation:
			tapped = s.addTap(c)
		default:
			fmt.Printf("Bad message recieved\n")
		}
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

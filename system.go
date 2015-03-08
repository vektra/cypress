package cypress

import (
	"fmt"
	"io"
	"net"
	"os"
	"strconv"
	"time"
)

const DefaultSocketPath = "/var/lib/cypress.sock"

var cMaxBuffer = 100

const cDefaultBacklog = 100

type Logger interface {
	Write(m *Message) error
	Close() error
}

type localConn struct {
	path      string
	conn      net.Conn
	connected bool
	feeder    chan *Message
	shutdown  chan struct{}
	done      chan struct{}
	buffer    []*Message
}

func ConnectTo(path string) Logger {
	_, err := os.Stat(path)
	if err != nil {
		panic(fmt.Errorf("log path is not available: %s", err))
	}

	backlog := cDefaultBacklog

	str := os.Getenv("LOG_BACKLOG")
	if str != "" {
		if i, err := strconv.Atoi(str); err == nil {
			backlog = i
		}
	}

	l := &localConn{
		path:     path,
		feeder:   make(chan *Message, backlog),
		shutdown: make(chan struct{}),
		done:     make(chan struct{}),
	}

	go l.process()

	return l
}

func Connect() Logger {
	return ConnectTo(LogPath())
}

func LogPath() string {
	path := os.Getenv("LOG_PATH")
	if path != "" {
		return path
	}

	return DefaultSocketPath
}

func (l *localConn) save(m *Message) {
	if len(l.buffer) >= cMaxBuffer {
		l.buffer = append([]*Message{m}, l.buffer[:cMaxBuffer-1]...)
	} else {
		l.buffer = append(l.buffer, m)
	}
}

var cPlus = []byte("+")

func WriteLocalMessage(w io.Writer, m *Message) error {
	enc := NewEncoder()

	_, err := enc.EncodeTo(m, w)
	return err
}

func (l *localConn) send(m *Message) error {
	return WriteLocalMessage(l.conn, m)
}

func (l *localConn) flush() {
	if !l.connected {
		conn, err := net.Dial("unix", l.path)

		if err != nil {
			return
		}

		l.conn = conn
		l.connected = true
	}

	for len(l.buffer) > 0 {
		m := l.buffer[0]

		err := l.send(m)
		if err == nil {
			l.buffer = l.buffer[1:]
		} else {
			l.conn.Close()
			l.connected = false
			break
		}
	}
}

const cMaxTries = 10

func (l *localConn) finalFlush() {
	tries := 0

start:
	tries++
	if !l.connected {
		conn, err := net.Dial("unix", l.path)

		if err != nil {
			if tries == cMaxTries {
				fmt.Fprintf(os.Stderr, "Unable to connect to local logger to flush\n")
				return
			}

			time.Sleep(1 * time.Second)
			goto start
		}

		l.conn = conn
		l.connected = true
	}

	for len(l.buffer) > 0 {
		m := l.buffer[0]

		err := l.send(m)
		if err == nil {
			l.buffer = l.buffer[1:]
		} else {
			l.conn.Close()
			l.connected = false

			time.Sleep(1 * time.Second)
			goto start
		}
	}
}

func (l *localConn) process() {
	tick := time.NewTicker(1 * time.Second)

	for {
		select {
		case m := <-l.feeder:
			// fast case
			if l.connected && l.send(m) == nil {
				continue
			}

			l.save(m)
			l.flush()

		case <-tick.C:
			l.flush()

		case <-l.shutdown:

			// drain any messages in feeder
		outside:
			for {
				select {
				case m := <-l.feeder:
					l.save(m)
				default:
					break outside
				}
			}

			l.finalFlush()
			l.done <- struct{}{}
			return
		}
	} // for(ever)
}

func (l *localConn) Close() error {
	l.shutdown <- struct{}{}
	<-l.done

	return nil
}

func (l *localConn) Write(m *Message) error {
	l.feeder <- m
	return nil
}

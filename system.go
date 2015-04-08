package cypress

import (
	"fmt"
	"net"
	"os"
	"strconv"
	"time"
)

// The path on this system that the agent listens
const DefaultSocketPath = "/var/lib/cypress.sock"

var cMaxBuffer = 100

const cDefaultBacklog = 100

// A simple interface used to represent a system logger
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
	enc       *Encoder
}

// Connect to an agent on a path and return a Logger
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

// Connect to the default system logger
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

// Buffer m until the logger returns
func (l *localConn) save(m *Message) {
	if len(l.buffer) >= cMaxBuffer {
		l.buffer = append([]*Message{m}, l.buffer[:cMaxBuffer-1]...)
	} else {
		l.buffer = append(l.buffer, m)
	}
}

// Send Message to the logger
func (l *localConn) send(m *Message) error {
	_, err := l.enc.Encode(m)
	return err
}

// Write out any buffered messages to the logger
func (l *localConn) flush() {
	if !l.connected {
		conn, err := net.Dial("unix", l.path)

		if err != nil {
			return
		}

		l.conn = conn
		l.connected = true
		l.enc = NewEncoder(conn)
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

// Write out any buffered messages to the logger before exitting
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
		l.enc = NewEncoder(conn)
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

// Work loop of the local connection. Reads messages, sends them to the
// logger, flushes the buffers, etc.
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

// Close the background proces goroutine
func (l *localConn) Close() error {
	l.shutdown <- struct{}{}
	<-l.done

	return nil
}

// Write the Message to the Logger
func (l *localConn) Write(m *Message) error {
	l.feeder <- m
	return nil
}

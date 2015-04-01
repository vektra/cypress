package tcplog

import (
	"net"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/vektra/cypress"
	"github.com/vektra/neko"
)

type TestFormatter struct{}

func (tf *TestFormatter) Format(m *cypress.Message) ([]byte, error) {
	return []byte(m.KVString()), nil
}

func TestRead(t *testing.T) {
	n := neko.Start(t)

	var l *Logger

	n.Setup(func() {
		l = NewLogger("", false, &TestFormatter{})
	})

	n.It("reads a byte slice", func() {
		ok := l.Read([]byte("This is a long line"))
		assert.NoError(t, ok)
	})

	n.It("reads a string", func() {
		ok := l.Read("This is a long line")
		assert.NoError(t, ok)
	})

	n.It("reads a cypress.Message", func() {
		message := NewMessage(t)
		ok := l.Read(message)
		assert.NoError(t, ok)
	})

	n.It("does not read an int", func() {
		ok := l.Read(1)
		assert.Error(t, ok)
	})

	n.Meow()
}

func TestWrite(t *testing.T) {
	n := neko.Start(t)

	var (
		l    *Logger
		line = []byte("This is a log line")
	)

	n.Setup(func() {
		l = NewLogger("", false, &TestFormatter{})
	})

	n.It("adds a log line to the pump", func() {
		l.write(line)

		select {
		case pumpLine := <-l.Pump:
			assert.Equal(t, line, pumpLine)

			var zero uint64 = 0

			assert.Equal(t, zero, l.PumpDropped)
		default:
			t.Fail()
		}
	})

	n.It("adds an error line to the pump if lines were dropped", func() {
		l.PumpDropped = 1
		l.write(line)

		select {
		case <-l.Pump:
			expected := "The tcplog pump dropped 1 log lines"
			actual := <-l.Pump

			assert.True(t, strings.Index(string(actual), expected) != -1)

			var zero uint64 = 0

			assert.Equal(t, zero, l.PumpDropped)
		default:
			t.Fail()
		}
	})

	n.It("does not add a log line and increments dropped counter if pump is full ", func() {
		l.Pump = make(chan []byte, 0)
		l.write(line)

		select {
		case <-l.Pump:
			t.Fail()
		default:
			var one uint64 = 1

			assert.Equal(t, one, l.PumpDropped)
		}
	})

	n.Meow()
}

func TestDial(t *testing.T) {
	s := NewTcpServer()

	go s.Run("127.0.0.1")

	l := NewLogger(<-s.Address, false, &TestFormatter{})

	conn, _ := l.dial()
	_, ok := conn.(net.Conn)
	defer conn.Close()

	assert.True(t, ok, "returns a connection")
}

func TestSendLogs(t *testing.T) {
	n := neko.Start(t)

	var (
		s    *TcpServer
		l    *Logger
		line = []byte("This is a log line")
		wg   sync.WaitGroup
	)

	n.Setup(func() {
		s = NewTcpServer()

		wg.Add(1)
		go func() {
			defer wg.Done()
			s.Run("127.0.0.1")
		}()

		l = NewLogger(<-s.Address, false, &TestFormatter{})

		wg.Add(1)
		go func() {
			defer wg.Done()
			l.sendLogs()
		}()
	})

	n.It("sends line from pipe to tcp server", func() {
		l.Pump <- line
		close(l.Pump)

		wg.Wait()

		select {
		case message := <-s.Messages:
			assert.Equal(t, string(line), string(message))
		default:
			t.Fail()
		}
	})

	n.Meow()
}

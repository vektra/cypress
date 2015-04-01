package tcplog

import (
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"sync/atomic"
	"time"

	"github.com/vektra/cypress"
)

const cBufferSize = 100

type Formatter interface {
	Format(m *cypress.Message) ([]byte, error)
}

type Logger struct {
	Formatter
	Address     string
	Ssl         bool
	Pump        chan []byte
	PumpClosed  bool
	PumpDropped uint64
	ConnDropped uint64
}

func NewLogger(address string, ssl bool, formatter Formatter) *Logger {
	return &Logger{
		Formatter:   formatter,
		Address:     address,
		Ssl:         ssl,
		Pump:        make(chan []byte, cBufferSize),
		PumpClosed:  false,
		PumpDropped: 0,
		ConnDropped: 0,
	}
}

func (l *Logger) Run() {
	l.sendLogs()
	defer l.cleanup()
}

func (l *Logger) Receive(message *cypress.Message) error {
	return l.Read(message)
}

func (l *Logger) Read(message interface{}) (err error) {
	var data []byte

	switch m := message.(type) {
	case []byte:
		data = m
	case string:
		data = []byte(m)
	case *cypress.Message:
		data, _ = l.Format(m)
	default:
		return errors.New("Unable to read message type")
	}

	return l.write(data)
}

func (l *Logger) write(line []byte) (err error) {
	if l.PumpClosed == true {
		return errors.New("Pump is closed")
	}

	select {
	case l.Pump <- line:
		if pumpDropped := atomic.LoadUint64(&l.PumpDropped); pumpDropped > 0 {
			logMessage := cypress.Log()
			logMessage.Add("error", fmt.Sprintf("The tcplog pump dropped %d log lines", pumpDropped))
			data, _ := l.Format(logMessage)

			select {
			case l.Pump <- data:
				atomic.AddUint64(&l.PumpDropped, -pumpDropped)
			default:
				return
			}
		}
	default:
		atomic.AddUint64(&l.PumpDropped, 1)
	}

	return nil
}

func (l *Logger) dial() (conn net.Conn, err error) {
	if l.Ssl == true {
		config := tls.Config{InsecureSkipVerify: true}
		conn, err = tls.Dial("tcp", l.Address, &config)
	} else {
		conn, err = net.Dial("tcp", l.Address)
	}
	return conn, err
}

func (l *Logger) sendLogs() {
	for {
		conn, err := l.dial()
		if err != nil {
			time.Sleep(1 * time.Second)
			continue // try to connect again
		}

		for {
			line, ok := <-l.Pump
			if ok != true {
				conn.Close()
				return // chan is closed, end processing
			}

			_, err = conn.Write(line)

			if err != nil {
				conn.Close()
				time.Sleep(1 * time.Second)

				conn, err = l.dial()
				if err != nil {
					atomic.AddUint64(&l.ConnDropped, 1)
					break // try to connect again
				}

				_, err = conn.Write(line)
				if err != nil {
					atomic.AddUint64(&l.ConnDropped, 1)
					break // try to connect again
				}
			}

			if connDropped := atomic.LoadUint64(&l.ConnDropped); connDropped > 0 {
				logMessage := cypress.Log()
				logMessage.Add("error", fmt.Sprintf("The tcplog connection dropped %d log lines", connDropped))
				data, _ := l.Format(logMessage)

				_, err = conn.Write(data)
				if err == nil {
					atomic.AddUint64(&l.ConnDropped, -connDropped)
				}
			}
		}

		conn.Close()
	}
}

func (l *Logger) cleanup() {
	if l.PumpClosed == false {
		close(l.Pump)
		l.PumpClosed = true
	}
}

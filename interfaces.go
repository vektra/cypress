package cypress

import (
	"errors"

	"github.com/vektra/tai64n"
)

type Receiver interface {
	Receive(msg *Message) error
}

type Generator interface {
	Generate() (*Message, error)
	Close() error
}

type GeneratorHandler interface {
	HandleGenerator(g Generator)
}

type Parser interface {
	Parse() error
}

type SendRequest interface {
	Ack(*Message)
	Nack(*Message)
}

type Sender interface {
	Send(m *Message, req SendRequest) error
}

type SendMaker interface {
	NewSender(window int) (Sender, error)
}

var ErrStopIteration = errors.New("stop iteration")

type LogHandlerFunc func(*Message) error

func (l LogHandlerFunc) HandleMessage(m *Message) error {
	return l(m)
}

func LogHandleFunc(f func(*Message) error) LogHandlerFunc {
	return LogHandlerFunc(f)
}

type LogHandler interface {
	HandleMessage(m *Message) error
}

type LogViewer interface {
	StreamIndex(index string, value interface{}, count uint64, h LogHandler) error
	TailIndex(index string, value interface{}, count uint64, h LogHandler) error

	StreamMatching(from *tai64n.TAI64N, crit Criteria, h LogHandler) error
	TailMatching(from *tai64n.TAI64N, crit Criteria, h LogHandler) error
}

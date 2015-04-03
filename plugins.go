package cypress

import (
	"errors"
	"strings"
)

var (
	ErrNoReceiver  = errors.New("no receiever available")
	ErrNoGenerator = errors.New("no generator available")
)

type Plugin interface {
	Receiver() (Receiver, error)
	Generator() (Generator, error)
}

var plugins = map[string]func() Plugin{}

func AddPlugin(name string, creator func() Plugin) {
	plugins[strings.ToLower(name)] = creator
}

func FindPlugin(name string) (Plugin, bool) {
	t, ok := plugins[strings.ToLower(name)]
	if !ok {
		return nil, false
	}

	return t(), true
}

type TestPlugin struct {
	Messages chan *Message
}

func (t *TestPlugin) Init() {
	t.Messages = make(chan *Message, 10)
}

func (t *TestPlugin) Generator() (Generator, error) {
	return t, nil
}

func (t *TestPlugin) Receiver() (Receiver, error) {
	return t, nil
}

func (t *TestPlugin) Generate() (*Message, error) {
	return <-t.Messages, nil
}

func (t *TestPlugin) Close() error {
	close(t.Messages)
	return nil
}

func (t *TestPlugin) Receive(m *Message) error {
	t.Messages <- m
	return nil
}

func init() {
	AddPlugin("Test", func() Plugin {
		t := &TestPlugin{}
		t.Init()
		return t
	})
}

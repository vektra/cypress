package cypress

import (
	"errors"
	"strings"
)

var (
	// Indicates that a Plugin does not have a Receiver
	ErrNoReceiver = errors.New("no receiver available")

	// Indicates that a Plugin does not have a Generator
	ErrNoGenerator = errors.New("no generator available")
)

// An interface implemented by plugins used by the router
type Plugin interface{}

type ReceiverPlugin interface {
	Receiver() (Receiver, error)
}

type GeneratorPlugin interface {
	Generator() (Generator, error)
}

type FiltererPlugin interface {
	Filterer() (Filterer, error)
}

var plugins = map[string]func() Plugin{}

// Add a new plugin by name with a function to create a new instance
// of this plugin.
func AddPlugin(name string, creator func() Plugin) {
	plugins[strings.ToLower(name)] = creator
}

// Find a plugin by name and invoke it's creator function to create
// a new Plugin instance
func FindPlugin(name string) (Plugin, bool) {
	t, ok := plugins[strings.ToLower(name)]
	if !ok {
		return nil, false
	}

	return t(), true
}

// Used for testing only
type TestPlugin struct {
	Messages     chan *Message
	FilterFields map[string]interface{}
}

func (t *TestPlugin) Init() {
	t.Messages = make(chan *Message, 10)
}

func (t *TestPlugin) Generator() (Generator, error) {
	return t, nil
}

func (t *TestPlugin) Generate() (*Message, error) {
	return <-t.Messages, nil
}

func (t *TestPlugin) Receiver() (Receiver, error) {
	return t, nil
}

func (t *TestPlugin) Receive(m *Message) error {
	t.Messages <- m
	return nil
}

func (t *TestPlugin) Filterer() (Filterer, error) {
	return t, nil
}

func (t *TestPlugin) Filter(m *Message) (*Message, error) {
	for k, v := range t.FilterFields {
		m.Add(k, v)
	}

	return m, nil
}

func (t *TestPlugin) Close() error {
	close(t.Messages)
	return nil
}

func init() {
	AddPlugin("Test", func() Plugin {
		t := &TestPlugin{}
		t.Init()
		return t
	})
}

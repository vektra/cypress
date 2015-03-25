package cypress

import "net"

type Plugin interface {
	Receiver() (Receiver, error)
	Generator() (Generator, error)
}

var plugins = map[string]func() Plugin{}

func AddPlugin(name string, creator func() Plugin) {
	plugins[name] = creator
}

func FindPlugin(name string) (Plugin, bool) {
	t, ok := plugins[name]
	if !ok {
		return nil, false
	}

	return t(), true
}

type TCPPlugin struct {
	Address  string
	Listener net.Listener
}

func (r *TCPPlugin) Receiver() (Receiver, error) {
	return NewTCPSend(r.Address, 0, DefaultTCPBuffer)
}

func (r *TCPPlugin) Generator() (Generator, error) {
	return NewTCPRecvGenerator(r.Address)
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
	AddPlugin("TCP", func() Plugin {
		return &TCPPlugin{}
	})

	AddPlugin("Test", func() Plugin {
		t := &TestPlugin{}
		t.Init()
		return t
	})
}

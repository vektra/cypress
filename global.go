package cypress

import "os"

func Available() bool {
	_, err := os.Stat(LogPath())
	return err == nil
}

type nullLogger struct{}

func (n *nullLogger) Write(m *Message) error {
	return nil
}

func (n *nullLogger) Close() error {
	return nil
}

var system Logger

func init() {
	system = &nullLogger{}
}

func Open() {
	system = Connect()
}

func Write(m *Message) error {
	return system.Write(m)
}

func Close() error {
	return system.Close()
}

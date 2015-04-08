package cypress

import "os"

// Indicate if the cypress agent is available
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

// Open a connection to the system logger
func Open() {
	system = Connect()
}

// Write a Message to the system logger
func Write(m *Message) error {
	return system.Write(m)
}

// Close the system logger
func Close() error {
	return system.Close()
}

package cypress

import "io"

// A Go channel that fits the Receiver and Generator interfaces.
type Channel chan *Message

// Return a message by reading from the channel.
func (c Channel) Generate() (*Message, error) {
	m, ok := <-c
	if !ok {
		return nil, io.EOF
	}

	return m, nil
}

// Write a message to the channel
func (c Channel) Receive(m *Message) error {
	c <- m
	return nil
}

// Close the channel down
func (c Channel) Close() error {
	close(c)
	return nil
}

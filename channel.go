package cypress

import "io"

type Channel chan *Message

func (c Channel) Generate() (*Message, error) {
	m, ok := <-c
	if !ok {
		return nil, io.EOF
	}

	return m, nil
}

func (c Channel) Receive(m *Message) error {
	c <- m
	return nil
}

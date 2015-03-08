package commands

import (
	"io"

	"github.com/vektra/cypress"
)

type Inject struct {
	r    io.Reader
	recv cypress.Receiver

	ss *cypress.SwitchStream
}

func NewInject(r io.Reader, recv cypress.Receiver) *Inject {
	ss := &cypress.SwitchStream{
		Input: r,
		Out:   recv,
	}

	return &Inject{r, recv, ss}
}

func (i *Inject) Run() error {
	return i.ss.Parse()
}

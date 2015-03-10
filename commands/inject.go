package commands

import (
	"io"

	"github.com/vektra/cypress"
)

type Inject struct {
	r    io.Reader
	recv cypress.Receiver

	dec *cypress.StreamDecoder
}

func NewInject(r io.Reader, recv cypress.Receiver) *Inject {
	dec := cypress.NewStreamDecoder(r)

	return &Inject{r, recv, dec}
}

func (i *Inject) Run() error {
	return cypress.Glue(i.dec, i.recv)
}

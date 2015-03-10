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

func NewInject(r io.Reader, recv cypress.Receiver) (*Inject, error) {
	dec, err := cypress.NewStreamDecoder(r)
	if err != nil {
		return nil, err
	}

	return &Inject{r, recv, dec}, nil
}

func (i *Inject) Run() error {
	return cypress.Glue(i.dec, i.recv)
}

package spool

import (
	"fmt"
	"os"

	"github.com/vektra/cypress"
	"github.com/vektra/cypress/cli/commands"
)

type Send struct {
	Dir string `short:"d" description:"where to write the messages to"`
}

func (s *Send) Execute(args []string) error {
	if s.Dir == "" {
		return fmt.Errorf("no target specified")
	}

	if _, err := os.Stat(s.Dir); err != nil {
		os.MkdirAll(s.Dir, 0755)
	}

	spool, err := NewSpool(s.Dir)
	if err != nil {
		return err
	}

	dec, err := cypress.NewStreamDecoder(os.Stdin)
	if err != nil {
		return err
	}

	return cypress.Glue(dec, spool)
}

type Recv struct {
	Dir string `short:"d" description:"where to write the messages to"`
}

func (r *Recv) Execute(args []string) error {
	if r.Dir == "" {
		return fmt.Errorf("no target specified")
	}

	if _, err := os.Stat(r.Dir); err != nil {
		return err
	}

	spool, err := NewSpool(r.Dir)
	if err != nil {
		return err
	}

	enc := cypress.NewStreamEncoder(os.Stdout)

	gen, err := spool.Generator()
	if err != nil {
		return err
	}

	return cypress.Glue(gen, enc)
}

func init() {
	commands.Add("spool:send", "write messages to a spool", "", &Send{})
	commands.Add("spool:recv", "read messages from a spool", "", &Recv{})
}

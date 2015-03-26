package spool

import (
	"fmt"
	"os"

	"github.com/vektra/cypress"
	"github.com/vektra/cypress/cli/commands"
)

type InjectCommand struct {
	Args struct {
		Target string `short:"s" positional-arg-name:"target" description:"where to write the messages to"`
	} `positional-args:"true"`
}

func (i *InjectCommand) Execute(args []string) error {
	dir := i.Args.Target
	if dir == "" {
		return fmt.Errorf("no target specified")
	}

	if _, err := os.Stat(dir); err != nil {
		os.MkdirAll(dir, 0755)
	}

	spool, err := NewSpool(dir)
	if err != nil {
		return err
	}

	dec, err := cypress.NewStreamDecoder(os.Stdin)
	if err != nil {
		return err
	}

	return cypress.Glue(dec, spool)
}

func init() {
	commands.Add("inject", "inject messages to a spool", "", &InjectCommand{})
}

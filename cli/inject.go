package cli

import (
	"fmt"
	"os"

	"github.com/vektra/cypress/commands"
	"github.com/vektra/cypress/plugins/spool"
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

	spool, err := spool.NewSpool(dir)
	if err != nil {
		return err
	}

	inj, err := commands.NewInject(os.Stdin, spool)
	if err != nil {
		return err
	}

	return inj.Run()
}

func init() {
	addCommand("inject", "inject messages", "", &InjectCommand{})
}

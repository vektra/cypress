package cli

import (
	"os"

	"github.com/mitchellh/cli"
	"github.com/vektra/cypress/commands"
	"github.com/vektra/cypress/plugin"
)

type InjectCommand struct {
	Ui cli.Ui
}

func (i *InjectCommand) Synopsis() string {
	return "inject messages"
}

func (i *InjectCommand) Help() string {
	return "inject messages"
}

func (i *InjectCommand) Run(args []string) int {
	dir := args[0]
	if dir == "" {
		return 1
	}

	if _, err := os.Stat(dir); err != nil {
		os.MkdirAll(dir, 0755)
	}

	spool, err := plugin.NewSpool(dir)
	if err != nil {
		return 1
	}

	inj, err := commands.NewInject(os.Stdin, spool)
	if err != nil {
		return 1
	}

	err = inj.Run()
	if err != nil {
		return 1
	}

	return 0
}

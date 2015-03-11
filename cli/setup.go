package cli

import (
	"os"

	"github.com/mitchellh/cli"
)

var Commands map[string]cli.CommandFactory

func init() {
	ui := &cli.BasicUi{Writer: os.Stdout}

	Commands = map[string]cli.CommandFactory{
		"cat": func() (cli.Command, error) {
			return &CatCommand{Ui: ui}, nil
		},
		"inject": func() (cli.Command, error) {
			return &InjectCommand{Ui: ui}, nil
		},
		"keys": func() (cli.Command, error) {
			return &KeysCommand{Ui: ui}, nil
		},
	}
}

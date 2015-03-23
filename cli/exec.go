package cli

import (
	"os"

	"github.com/vektra/cypress"
)

type Exec struct{}

func (e *Exec) Execute(args []string) error {
	r, err := cypress.NewRun(args[0], args[1:]...)
	if err != nil {
		return err
	}

	enc := cypress.NewStreamEncoder(os.Stdout)

	return cypress.Glue(r, enc)
}

func init() {
	addCommand("exec", "execute a command and emit a message stream", "", &Exec{})
}

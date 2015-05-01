package file

import (
	"fmt"
	"io"
	"log"
	"os"

	"github.com/vektra/cypress"
	"github.com/vektra/cypress/cli/commands"
)

type CLI struct {
	Once bool   `short:"o" long:"once" description:"Read the file once, don't follow it"`
	DB   string `short:"d" long:"offset-db" description:"Track file offsets and use them"`

	Debug bool `long:"debug" description:"Output debug information to stderr"`

	output io.WriteCloser
}

var dbgLog = log.New(os.Stderr, "cypress-file: ", log.LstdFlags)

func (c *CLI) Execute(args []string) error {
	var err error

	if len(args) == 0 {
		return fmt.Errorf("provide at least one file path")
	}

	m := NewMonitor()
	m.Debug = c.Debug

	if c.DB != "" {
		err = m.OpenOffsetDB(c.DB)
		if err != nil {
			return err
		}

		commands.OnShutdown(m.WaitShutdown)
	}

	err = m.OpenFiles(c.Once, args)
	if err != nil {
		return err
	}

	out := c.output
	if out == nil {
		out = os.Stdout
	}

	enc := cypress.NewStreamEncoder(out)

	return m.Run(enc)
}

func init() {
	commands.Add("file", "read files from a file", "", &CLI{})
}

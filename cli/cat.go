package cli

import (
	"fmt"
	"io"
	"os"

	"github.com/vektra/cypress/commands"
	"github.com/vektra/cypress/plugin"
)

type CatCommand struct {
	Kv     bool `short:"k" description:"display in key=value format"`
	Human  bool `short:"H" description:"display in easy to read format"`
	Json   bool `short:"j" description:"display in json"`
	Native bool `short:"n" description:"output as native binary"`
}

func (c *CatCommand) Execute(args []string) error {
	var cnt int

	if c.Kv {
		cnt++
	}

	if c.Human {
		cnt++
	}

	if c.Json {
		cnt++
	}

	if c.Native {
		cnt++
	}

	var format commands.Format

	switch {
	case cnt == 0:
		format = commands.KV
	case cnt > 1:
		return fmt.Errorf("multiple display types requested, only use one")
	case c.Kv:
		format = commands.KV
	case c.Human:
		format = commands.HUMAN
	case c.Json:
		format = commands.JSON
	case c.Native:
		format = commands.NATIVE
	}

	dir := args[0]
	if dir == "" {
		return fmt.Errorf("no source specified")
	}

	spool, err := plugin.NewSpool(dir)
	if err != nil {
		return err
	}

	gen, err := spool.Generator()
	if err != nil {
		return err
	}

	cat, err := commands.NewCat(os.Stdout, gen, format)
	if err != nil {
		return err
	}

	err = cat.Run()
	if err != nil {
		if err == io.EOF {
			return nil
		}

		return err
	}

	return nil
}

func init() {
	long := `Given a source of messages, the cat command will read those messages in and print them out. Commonly, users point cat at a spool directory to read the message contained within.`

	addCommand("cat", "display messages", long, &CatCommand{})
}

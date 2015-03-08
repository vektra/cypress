package cli

import (
	"flag"
	"io"
	"os"
	"strings"

	"github.com/mitchellh/cli"
	"github.com/vektra/cypress/commands"
	"github.com/vektra/cypress/plugin"
)

type CatCommand struct {
	Ui cli.Ui

	kv     bool
	human  bool
	json   bool
	native bool
}

func (c *CatCommand) Synopsis() string {
	return "print out message"
}

func (c *CatCommand) Help() string {
	helptext := `
Usage: cypress cat [options]

  Read from a message source and print out messages within that
	source.

Options:

  -kv      Display messages in native text (kv, or key-value) format
  -human   Display messages in an easy to read format
	-json    Display messages in JSON
	-native  Output messages as binary in native format. This is mostly
	         useful for pulling messages out of a source and piping
					 them through cypress to something else, like grep.
	`

	return strings.TrimSpace(helptext)
}

func (c *CatCommand) Run(args []string) int {
	cmdFlags := flag.NewFlagSet("cat", flag.ContinueOnError)
	cmdFlags.Usage = func() { c.Ui.Output(c.Help()) }
	cmdFlags.BoolVar(&c.kv, "kv", false, "")
	cmdFlags.BoolVar(&c.human, "human", false, "")
	cmdFlags.BoolVar(&c.json, "json", false, "")
	cmdFlags.BoolVar(&c.native, "native", false, "")

	err := cmdFlags.Parse(args)
	if err != nil {
		return 1
	}

	var cnt int

	if c.kv {
		cnt++
	}

	if c.human {
		cnt++
	}

	if c.json {
		cnt++
	}

	if c.native {
		cnt++
	}

	var format commands.Format

	switch {
	case cnt == 0:
		format = commands.KV
	case cnt > 1:
		return 1
	case c.kv:
		format = commands.KV
	case c.human:
		format = commands.HUMAN
	case c.json:
		format = commands.JSON
	case c.native:
		format = commands.NATIVE
	}

	dir := cmdFlags.Arg(0)
	if dir == "" {
		return 1
	}

	spool, err := plugin.NewSpool(dir)
	if err != nil {
		return 1
	}

	gen, err := spool.Generator()
	if err != nil {
		return 1
	}

	cat, err := commands.NewCat(os.Stdout, gen, format)
	if err != nil {
		return 1
	}

	err = cat.Run()
	if err != nil {
		if err == io.EOF {
			return 0
		}
		return 1
	}

	return 0
}

package cli

import (
	"fmt"

	"github.com/vektra/cypress/cli/commands"
)

import (
	"sync"

	"github.com/jessevdk/go-flags"
)

var (
	globalParser      *flags.Parser
	globalParserSetup sync.Once
)

func parser() *flags.Parser {
	globalParserSetup.Do(func() {
		globalParser = flags.NewNamedParser("cypress", flags.Default|flags.PassAfterNonOption)
	})

	return globalParser
}

func AddCommand(name, short, long string, cmd interface{}) {
	_, err := parser().AddCommand(name, short, long, cmd)
	if err != nil {
		panic(err)
	}
}

func addCommand(name, short, long string, cmd interface{}) {
	AddCommand(name, short, long, cmd)
}

func Run(args []string) int {
	commands.SetShutdownHandler(Lifecycle)

	for _, cmd := range commands.Commands {
		AddCommand(cmd.Name, cmd.Short, cmd.Long, cmd.Cmd)
	}

	defer Lifecycle.RunCleanup()

	Lifecycle.Start()

	_, err := parser().Parse()
	if err != nil {
		if ferr, ok := err.(*flags.Error); ok {
			if ferr.Type == flags.ErrCommandRequired {
				return 1
			}

			if ferr.Type == flags.ErrHelp {
				return 1
			}
		}

		fmt.Printf("Error: %s\n", err)
		return 1
	}

	return 0
}

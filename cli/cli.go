package cli

import "fmt"

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

func addCommand(name, short, long string, cmd interface{}) {
	_, err := parser().AddCommand(name, short, long, cmd)
	if err != nil {
		panic(err)
	}
}

func Run(args []string) int {
	defer Lifecycle.RunCleanup()

	Lifecycle.Start()

	_, err := parser().Parse()
	if err != nil {
		fmt.Printf("Error: %s\n", err)
		return 1
	}

	return 0
}

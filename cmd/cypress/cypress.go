package main

import (
	"os"

	"github.com/vektra/cypress/cli"
	_ "github.com/vektra/cypress/plugins/all"
	_ "github.com/vektra/cypress/tools"
)

func main() {
	os.Exit(cli.Run(os.Args[1:]))
}

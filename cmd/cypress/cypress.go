package main

import (
	"os"

	"github.com/vektra/cypress/cli"
)

func main() {
	os.Exit(cli.Run(os.Args[1:]))
}

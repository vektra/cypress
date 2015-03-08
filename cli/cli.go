package cli

import (
	"log"
	"os"

	"github.com/mitchellh/cli"
)

func Run(args []string) int {
	c := cli.NewCLI("app", "0.2.0")
	c.Args = os.Args[1:]
	c.Commands = Commands

	status, err := c.Run()
	if err != nil {
		log.Println(err)
	}

	return status
}

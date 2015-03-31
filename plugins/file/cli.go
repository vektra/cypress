package file

import (
	"fmt"
	"os"

	"github.com/vektra/cypress"
	"github.com/vektra/cypress/cli/commands"
)

type CLI struct {
	Once bool   `short:"o" long:"once" description:"Read the file once, don't follow it"`
	DB   string `short:"d" long:"offset-db" description:"Track file offsets and use them"`
}

func (c *CLI) Execute(args []string) error {
	var err error

	if len(args) == 0 {
		return fmt.Errorf("provide at least one file path")
	}

	var db *OffsetDB

	if c.DB != "" {
		db, err = NewOffsetDB(c.DB)
		if err != nil {
			return err
		}
	}

	msgs := make(chan *cypress.Message, len(args))

	for _, path := range args {
		var offset int64
		var f *File

		if c.DB != "" {
			entry, err := db.Get(path)
			if err != nil {
				return err
			}

			if entry != nil && entry.Valid() {
				offset = entry.Offset
			}

			commands.OnShutdown(func() {
				offset, err := f.Tell()
				if err == nil {
					db.Set(path, offset)
				}
			})
		}

		if c.Once {
			f, err = NewFile(path, offset)
		} else {
			f, err = NewFollowFile(path, offset)
		}

		if err != nil {
			return err
		}

		go func() {
			for {
				m, err := f.Generate()
				if err != nil {
					return
				}

				msgs <- m
			}
		}()
	}

	enc := cypress.NewStreamEncoder(os.Stdout)

	for m := range msgs {
		err := enc.Receive(m)
		if err != nil {
			return err
		}
	}

	return nil
}

func init() {
	commands.Add("file", "read files from a file", "", &CLI{})
}

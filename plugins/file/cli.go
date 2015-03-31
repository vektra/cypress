package file

import (
	"fmt"
	"io"
	"os"
	"sync"

	"github.com/vektra/cypress"
	"github.com/vektra/cypress/cli/commands"
)

type CLI struct {
	Once bool   `short:"o" long:"once" description:"Read the file once, don't follow it"`
	DB   string `short:"d" long:"offset-db" description:"Track file offsets and use them"`

	output io.Writer
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

	var wg sync.WaitGroup

	var files []*File

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
		}

		if c.Once {
			f, err = NewFile(path, offset)
		} else {
			f, err = NewFollowFile(path, offset)
		}

		if err != nil {
			return err
		}

		if db != nil {
			thisPath := path
			commands.OnShutdown(func() {
				offset, err := f.Tell()
				if err == nil {
					db.Set(thisPath, offset)
				}
			})
		}

		files = append(files, f)

		wg.Add(1)
		go func(path string) {
			defer wg.Done()

			if db != nil {
				defer func() {
					offset, err := f.Tell()
					if err == nil {
						err = db.Set(path, offset)
						if err != nil {
							fmt.Fprintf(os.Stderr, "Error updating offsetdb: %s\n", err)
						}
					}
				}()
			}

			for {
				m, err := f.Generate()
				if err != nil {
					return
				}

				msgs <- m
			}
		}(path)
	}

	go func() {
		wg.Wait()
		close(msgs)
	}()

	out := c.output
	if out == nil {
		out = os.Stdout
	}

	enc := cypress.NewStreamEncoder(out)

	for m := range msgs {
		err := enc.Receive(m)
		if err != nil {
			for _, f := range files {
				f.Close()
			}

			wg.Wait()
			return err
		}
	}

	return nil
}

func init() {
	commands.Add("file", "read files from a file", "", &CLI{})
}

package tools

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/vektra/cypress"
	"github.com/vektra/cypress/cli/commands"
)

type Format int

const (
	KV Format = iota
	HUMAN
	JSON
	NATIVE
)

type Cat struct {
	out io.Writer
	gen cypress.Generator

	format Format

	buf []byte
}

func NewCat(out io.Writer, gen cypress.Generator, format Format) (*Cat, error) {
	return &Cat{out, gen, format, make([]byte, 128)}, nil
}

var nl = []byte{'\n'}

func (c *Cat) Run() error {
	defer c.gen.Close()

	var (
		buf bytes.Buffer
		enc *cypress.StreamEncoder
	)

	if c.format == NATIVE {
		enc = cypress.NewStreamEncoder(c.out)
	}

	for {
		m, err := c.gen.Generate()
		if err != nil {
			return err
		}

		if m == nil {
			break
		}

		switch c.format {
		case KV:
			buf.Reset()

			m.KVStringInto(&buf)
			c.out.Write(buf.Bytes())
			c.out.Write(nl)
		case HUMAN:
			c.out.Write([]byte(m.HumanString()))
			c.out.Write(nl)
		case JSON:
			json.NewEncoder(c.out).Encode(m)
		case NATIVE:
			err := enc.Receive(m)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

type CatCLI struct {
	Kv     bool `short:"k" description:"display in key=value format"`
	Human  bool `short:"H" description:"display in easy to read format"`
	Json   bool `short:"j" description:"display in json"`
	Native bool `short:"n" description:"output as native binary"`
}

func (c *CatCLI) Execute(args []string) error {
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

	var format Format

	switch {
	case cnt == 0:
		format = KV
	case cnt > 1:
		return fmt.Errorf("multiple display types requested, only use one")
	case c.Kv:
		format = KV
	case c.Human:
		format = HUMAN
	case c.Json:
		format = JSON
	case c.Native:
		format = NATIVE
	}

	dec, err := cypress.NewStreamDecoder(os.Stdin)
	if err != nil {
		return err
	}

	cat, err := NewCat(os.Stdout, dec, format)
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
	long := `Given a stream on stdin, the cat command will read those messages in and print them out.`

	commands.Add("cat", "display messages", long, &CatCLI{})
}

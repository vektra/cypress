package commands

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"io"

	"github.com/vektra/cypress"
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

	var buf bytes.Buffer

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
			bytes, err := m.Marshal()
			if err != nil {
				return err
			}

			binary.BigEndian.PutUint64(c.buf[:8], uint64(len(bytes)))
			c.out.Write(c.buf[:8])
			c.out.Write(bytes)
		}
	}

	return nil
}

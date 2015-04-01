package file

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vektra/cypress"
	"github.com/vektra/neko"
)

func TestCLI(t *testing.T) {
	n := neko.Start(t)

	tmpdir, err := ioutil.TempDir("", "file")
	require.NoError(t, err)

	defer os.RemoveAll(tmpdir)

	n.It("generates a stream from lines in a file", func() {
		var buf bytes.Buffer

		file := filepath.Join(tmpdir, "blah.log")

		f, err := os.Create(file)
		require.NoError(t, err)

		defer f.Close()
		defer os.Remove(file)

		fmt.Fprintf(f, "from the file\n")

		f.Close()

		cli := &CLI{Once: true, output: &buf}

		err = cli.Execute([]string{file})
		require.NoError(t, err)

		dec, err := cypress.NewStreamDecoder(&buf)
		require.NoError(t, err)

		m, err := dec.Generate()
		require.NoError(t, err)

		src, ok := m.GetTag("source")
		require.True(t, ok)

		assert.Equal(t, "blah.log", src)

		msg, ok := m.Get("message")
		require.True(t, ok)

		assert.Equal(t, "from the file", msg)
	})

	n.It("uses the offsetdb if requested", func() {
		var buf bytes.Buffer

		file := filepath.Join(tmpdir, "blah.log")

		f, err := os.Create(file)
		require.NoError(t, err)

		defer f.Close()
		defer os.Remove(file)

		fmt.Fprintf(f, "from the file\n")

		dbpath := filepath.Join(tmpdir, "db")

		cli := &CLI{Once: true, DB: dbpath, output: &buf}

		err = cli.Execute([]string{file})
		require.NoError(t, err)

		dec, err := cypress.NewStreamDecoder(&buf)
		require.NoError(t, err)

		m, err := dec.Generate()
		require.NoError(t, err)

		src, ok := m.GetTag("source")
		require.True(t, ok)

		assert.Equal(t, "blah.log", src)

		msg, ok := m.Get("message")
		require.True(t, ok)

		assert.Equal(t, "from the file", msg)

		// add another line

		fmt.Fprint(f, "this is another line\n")

		cli = &CLI{Once: true, DB: dbpath, output: &buf}

		err = cli.Execute([]string{file})
		require.NoError(t, err)

		m, err = dec.Generate()
		require.NoError(t, err)

		src, ok = m.GetTag("source")
		require.True(t, ok)

		assert.Equal(t, "blah.log", src)

		msg, ok = m.Get("message")
		require.True(t, ok)

		assert.Equal(t, "this is another line", msg)
	})

	n.Meow()
}
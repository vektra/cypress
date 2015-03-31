package file

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vektra/neko"
)

func TestFile(t *testing.T) {
	n := neko.Start(t)

	tmpdir, err := ioutil.TempDir("", "file")
	require.NoError(t, err)

	defer os.RemoveAll(tmpdir)

	n.It("generates messages for each line in a file", func() {
		path := filepath.Join(tmpdir, "blah.log")

		f, err := os.Create(path)
		require.NoError(t, err)

		defer f.Close()
		defer os.Remove(path)

		fmt.Fprint(f, "line 1 has stuff\nline 2 as well\n")

		fo, err := NewFile(path, 0)
		require.NoError(t, err)

		m1, err := fo.Generate()
		require.NoError(t, err)

		msg1, ok := m1.GetString("message")
		require.True(t, ok)

		assert.Equal(t, "line 1 has stuff", msg1)

		m2, err := fo.Generate()
		require.NoError(t, err)

		msg2, ok := m2.GetString("message")
		require.True(t, ok)

		assert.Equal(t, "line 2 as well", msg2)

		_, err = fo.Generate()
		require.Equal(t, io.EOF, err)
	})

	n.It("generates messages as new lines are added", func() {
		path := filepath.Join(tmpdir, "blah.log")

		f, err := os.Create(path)
		require.NoError(t, err)

		defer f.Close()
		defer os.Remove(path)

		fmt.Fprint(f, "line 1 has stuff\n")

		fo, err := NewFollowFile(path, 0)
		require.NoError(t, err)

		m1, err := fo.Generate()
		require.NoError(t, err)

		msg1, ok := m1.GetString("message")
		require.True(t, ok)

		assert.Equal(t, "line 1 has stuff", msg1)

		fmt.Fprint(f, "line 2 comes later\n")

		m2, err := fo.Generate()
		require.NoError(t, err)

		msg2, ok := m2.GetString("message")
		require.True(t, ok)

		assert.Equal(t, "line 2 comes later", msg2)
	})

	n.It("reopens a file that is deleted", func() {
		path := filepath.Join(tmpdir, "blah.log")

		f, err := os.Create(path)
		require.NoError(t, err)

		defer f.Close()
		defer os.Remove(path)

		fmt.Fprint(f, "line 1 has stuff\n")

		fo, err := NewFollowFile(path, 0)
		require.NoError(t, err)

		m1, err := fo.Generate()
		require.NoError(t, err)

		msg1, ok := m1.GetString("message")
		require.True(t, ok)

		assert.Equal(t, "line 1 has stuff", msg1)

		f.Close()
		os.Remove(path)

		f, err = os.Create(path)
		require.NoError(t, err)

		defer f.Close()
		defer os.Remove(path)

		fmt.Fprint(f, "line 2 comes in another file\n")

		m2, err := fo.Generate()
		require.NoError(t, err)

		msg2, ok := m2.GetString("message")
		require.True(t, ok)

		assert.Equal(t, "line 2 comes in another file", msg2)
	})

	n.It("can continue reading data after a certain offset", func() {
		path := filepath.Join(tmpdir, "blah.log")

		f, err := os.Create(path)
		require.NoError(t, err)

		defer f.Close()
		defer os.Remove(path)

		fmt.Fprint(f, "line 1 has stuff\n")

		fo, err := NewFile(path, 0)
		require.NoError(t, err)

		m1, err := fo.Generate()
		require.NoError(t, err)

		msg1, ok := m1.GetString("message")
		require.True(t, ok)

		assert.Equal(t, "line 1 has stuff", msg1)

		err = fo.Close()
		require.NoError(t, err)

		offset, err := fo.Tell()
		require.NoError(t, err)

		fmt.Fprint(f, "line 2 after the break\n")

		fo, err = NewFile(path, offset)
		require.NoError(t, err)

		m2, err := fo.Generate()
		require.NoError(t, err)

		msg2, ok := m2.GetString("message")
		require.True(t, ok)

		assert.Equal(t, "line 2 after the break", msg2)

		_, err = fo.Generate()
		require.Equal(t, io.EOF, err)
	})

	n.It("can continue reading data after a certain offset when following", func() {
		path := filepath.Join(tmpdir, "blah.log")

		f, err := os.Create(path)
		require.NoError(t, err)

		defer f.Close()
		defer os.Remove(path)

		fmt.Fprint(f, "line 1 has stuff\n")

		fo, err := NewFollowFile(path, 0)
		require.NoError(t, err)

		m1, err := fo.Generate()
		require.NoError(t, err)

		msg1, ok := m1.GetString("message")
		require.True(t, ok)

		assert.Equal(t, "line 1 has stuff", msg1)

		err = fo.Close()
		require.NoError(t, err)

		offset, err := fo.Tell()
		require.NoError(t, err)

		fmt.Fprint(f, "line 2 after the break\n")

		fo, err = NewFollowFile(path, offset)
		require.NoError(t, err)

		m2, err := fo.Generate()
		require.NoError(t, err)

		msg2, ok := m2.GetString("message")
		require.True(t, ok)

		assert.Equal(t, "line 2 after the break", msg2)
	})
	n.Meow()
}
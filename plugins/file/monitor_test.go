package file

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vektra/cypress"
	"github.com/vektra/neko"
)

func TestMonitor(t *testing.T) {
	n := neko.Start(t)

	tmpdir, err := ioutil.TempDir("", "monitor")
	require.NoError(t, err)

	defer os.RemoveAll(tmpdir)

	file1 := filepath.Join(tmpdir, "foo")
	file2 := filepath.Join(tmpdir, "bar")

	files := []string{file1, file2}

	dbpath := filepath.Join(tmpdir, "offsets")

	n.Setup(func() {
		f, err := os.Create(file1)
		require.NoError(t, err)

		fmt.Fprintf(f, "foo has lines\n")

		f.Close()

		f, err = os.Create(file2)
		require.NoError(t, err)

		fmt.Fprintf(f, "bar has data\n")

		f.Close()
	})

	n.Cleanup(func() {
		os.RemoveAll(dbpath)
		os.Remove(file1)
		os.Remove(file2)
	})

	n.It("watches multiple files and outputs their lines", func() {
		m := NewMonitor()

		var buf cypress.BufferReceiver

		err := m.OpenFiles(true, files)
		require.NoError(t, err)

		err = m.Run(&buf)
		require.NoError(t, err)

		var msgs []string

		for _, m := range buf.Messages {
			str, ok := m.GetString("message")
			require.True(t, ok)

			msgs = append(msgs, str)
		}

		sort.Strings(msgs)

		assert.Equal(t, "bar has data", msgs[0])
		assert.Equal(t, "foo has lines", msgs[1])
	})

	n.It("tracks and uses offsets if requested", func() {
		m := NewMonitor()

		var buf cypress.BufferReceiver

		err = m.OpenOffsetDB(dbpath)
		require.NoError(t, err)

		err := m.OpenFiles(true, files)
		require.NoError(t, err)

		err = m.Run(&buf)
		require.NoError(t, err)

		var msgs []string

		for _, m := range buf.Messages {
			str, ok := m.GetString("message")
			require.True(t, ok)

			msgs = append(msgs, str)
		}

		sort.Strings(msgs)

		assert.Equal(t, "bar has data", msgs[0])
		assert.Equal(t, "foo has lines", msgs[1])

		db, err := NewOffsetDB(dbpath)
		require.NoError(t, err)

		entry, err := db.Get(file1)
		require.NoError(t, err)

		require.NotNil(t, entry)

		assert.True(t, entry.Offset > 0)

		f, err := os.OpenFile(file1, os.O_APPEND|os.O_WRONLY, 0644)
		require.NoError(t, err)

		fmt.Fprint(f, "more data, more fun\n")

		f.Close()

		m = NewMonitor()

		var buf2 cypress.BufferReceiver

		err = m.OpenOffsetDB(dbpath)
		require.NoError(t, err)

		err = m.OpenFiles(true, files)
		require.NoError(t, err)

		err = m.Run(&buf2)
		require.NoError(t, err)

		msg := buf2.Messages[0]

		str, ok := msg.GetString("message")
		require.True(t, ok)

		assert.Equal(t, "more data, more fun", str)
	})

	n.Meow()
}

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
	"github.com/vektra/neko"
)

func TestPlugin(t *testing.T) {
	n := neko.Start(t)

	tmpdir, err := ioutil.TempDir("", "monitor")
	require.NoError(t, err)

	defer os.RemoveAll(tmpdir)

	file1 := filepath.Join(tmpdir, "foo2")
	file2 := filepath.Join(tmpdir, "foo1")

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

	n.It("sets up a generator for messages from lines", func() {
		plugin := &Plugin{
			Paths: []string{tmpdir + "/foo*"},
		}

		gen, err := plugin.Generator()
		require.NoError(t, err)

		var msgs []string

		m1, err := gen.Generate()
		require.NoError(t, err)

		msg, ok := m1.GetString("message")
		require.True(t, ok)

		msgs = append(msgs, msg)

		m2, err := gen.Generate()
		require.NoError(t, err)

		msg, ok = m2.GetString("message")
		require.True(t, ok)

		msgs = append(msgs, msg)

		sort.Strings(msgs)

		assert.Equal(t, "bar has data", msgs[0])
		assert.Equal(t, "foo has lines", msgs[1])
	})

	n.It("uses an offset db if available", func() {
		db, err := NewOffsetDB(dbpath)
		require.NoError(t, err)

		err = db.Set(file1, 4)
		require.NoError(t, err)

		plugin := &Plugin{
			Paths:    []string{tmpdir + "/foo*"},
			OffsetDB: dbpath,
		}

		gen, err := plugin.Generator()
		require.NoError(t, err)

		var msgs []string

		m1, err := gen.Generate()
		require.NoError(t, err)

		msg, ok := m1.GetString("message")
		require.True(t, ok)

		msgs = append(msgs, msg)

		m2, err := gen.Generate()
		require.NoError(t, err)

		msg, ok = m2.GetString("message")
		require.True(t, ok)

		msgs = append(msgs, msg)

		sort.Strings(msgs)

		assert.Equal(t, "bar has data", msgs[0])
		assert.Equal(t, "has lines", msgs[1])
	})

	n.Meow()
}

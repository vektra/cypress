package plugin

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vektra/cypress"
	"github.com/vektra/neko"
)

func TestSpool(t *testing.T) {
	n := neko.Start(t)

	root, err := ioutil.TempDir("", "spool")
	require.NoError(t, err)

	defer os.RemoveAll(root)

	tmpdir := filepath.Join(root, "spool")

	var sf *Spool

	n.Setup(func() {
		os.Mkdir(tmpdir, 0755)
		var err error
		sf, err = NewSpool(tmpdir)

		require.NoError(t, err)
	})

	n.Cleanup(func() {
		os.RemoveAll(tmpdir)
	})

	n.It("writes message to the current file in the spool dir", func() {
		m := cypress.Log()
		m.Add("hello", "world")

		err := sf.Receive(m)
		require.NoError(t, err)

		f, err := os.Open(filepath.Join(tmpdir, "current"))
		require.NoError(t, err)

		sd, err := cypress.NewStreamDecoder(f)
		require.NoError(t, err)

		m2, err := sd.Generate()
		require.NoError(t, err)

		assert.Equal(t, m.GetTimestamp(), m2.GetTimestamp())
		subject, ok := m2.GetString("hello")
		require.True(t, ok)

		assert.Equal(t, "world", subject)
	})

	n.It("provides a generator for reading messages back from the spool", func() {
		m := cypress.Log()
		m.Add("hello", "world")

		err := sf.Receive(m)
		require.NoError(t, err)

		gen, err := sf.Generator()
		require.NoError(t, err)

		m2, err := gen.Generate()
		require.NoError(t, err)

		defer gen.Close()

		assert.Equal(t, m.GetTimestamp(), m2.GetTimestamp())
		subject, ok := m2.GetString("hello")
		require.True(t, ok)

		assert.Equal(t, "world", subject)
	})

	n.Only("reads all files when generating messages from the spool", func() {
		m := cypress.Log()
		m.Add("source", "old")

		err := sf.Receive(m)
		require.NoError(t, err)

		err = sf.Rotate()
		require.NoError(t, err)

		cm := cypress.Log()
		cm.Add("source", "current")

		err = sf.Receive(cm)
		require.NoError(t, err)

		s, err := NewSpool(tmpdir)
		require.NoError(t, err)

		gen, err := s.Generator()
		require.NoError(t, err)

		m2, err := gen.Generate()
		require.NoError(t, err)

		defer gen.Close()

		assert.Equal(t, m.GetTimestamp(), m2.GetTimestamp())
		subject, ok := m2.GetString("source")
		require.True(t, ok)

		assert.Equal(t, "old", subject)

		m3, err := gen.Generate()
		require.NoError(t, err)

		source, ok := m3.GetString("source")
		require.True(t, ok)

		assert.Equal(t, "current", source)
	})

	n.Meow()
}

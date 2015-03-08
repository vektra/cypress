package plugin

import (
	"encoding/binary"
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

		err := sf.Read(m)
		require.NoError(t, err)

		data, err := ioutil.ReadFile(filepath.Join(tmpdir, "current"))
		require.NoError(t, err)

		assert.Equal(t, byte('+'), data[0])

		marsh, err := m.Marshal()
		require.NoError(t, err)

		l := binary.BigEndian.Uint64(data[1:])
		assert.Equal(t, l, uint64(len(marsh)))

		m2 := &cypress.Message{}

		err = m2.Unmarshal(data[9 : 9+l])
		require.NoError(t, err)

		assert.Equal(t, m.GetTimestamp(), m2.GetTimestamp())
		subject, ok := m2.GetString("hello")
		require.True(t, ok)

		assert.Equal(t, "world", subject)
	})

	n.It("provides a generator for reading messages back from the spool", func() {
		m := cypress.Log()
		m.Add("hello", "world")

		err := sf.Read(m)
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

	n.Meow()
}

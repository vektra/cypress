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

func TestSpoolFile(t *testing.T) {
	n := neko.Start(t)

	tmpdir, err := ioutil.TempDir("", "spool")
	require.NoError(t, err)

	defer os.RemoveAll(tmpdir)

	var sf *SpoolFile

	n.Setup(func() {
		var err error
		sf, err = NewSpoolFile(tmpdir)

		require.NoError(t, err)
	})

	n.It("writes message to the current file in the spool dir", func() {
		m := cypress.Log()
		m.Add("hello", "world")

		err := sf.Read(m)
		require.NoError(t, err)

		data, err := ioutil.ReadFile(filepath.Join(tmpdir, "current"))
		require.NoError(t, err)

		marsh, err := m.Marshal()
		require.NoError(t, err)

		l := binary.BigEndian.Uint64(data)
		assert.Equal(t, l, uint64(len(marsh)))

		m2 := &cypress.Message{}

		err = m2.Unmarshal(data[8:])
		require.NoError(t, err)

		assert.Equal(t, m.GetTimestamp(), m2.GetTimestamp())
		subject, ok := m2.GetString("hello")
		require.True(t, ok)

		assert.Equal(t, "world", subject)
	})

	n.Meow()
}

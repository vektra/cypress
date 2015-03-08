package commands

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vektra/cypress"
	"github.com/vektra/neko"
)

func TestCat(t *testing.T) {
	n := neko.Start(t)

	n.It("writes out messages in the KV format", func() {
		var out bytes.Buffer

		m := cypress.Log()
		m.Add("hello", "world")

		gen := cypress.StaticGenerator(m)

		cat, err := NewCat(&out, gen, KV)
		require.NoError(t, err)

		err = cat.Run()
		require.NoError(t, err)

		assert.Equal(t, m.KVString()+"\n", out.String())
	})

	n.It("writes out messages in the JSON format", func() {
		var out bytes.Buffer

		m := cypress.Log()
		m.Add("hello", "world")

		gen := cypress.StaticGenerator(m)

		cat, err := NewCat(&out, gen, JSON)
		require.NoError(t, err)

		err = cat.Run()
		require.NoError(t, err)

		str, err := json.Marshal(m)
		require.NoError(t, err)

		assert.Equal(t, string(str)+"\n", out.String())
	})

	n.It("writes out messages in the native format", func() {
		var out bytes.Buffer

		m := cypress.Log()
		m.Add("hello", "world")

		gen := cypress.StaticGenerator(m)

		cat, err := NewCat(&out, gen, NATIVE)
		require.NoError(t, err)

		err = cat.Run()
		require.NoError(t, err)

		bytes, err := m.Marshal()
		require.NoError(t, err)

		szbuf := make([]byte, 8)

		binary.BigEndian.PutUint64(szbuf, uint64(len(bytes)))

		assert.Equal(t, string(szbuf)+string(bytes), out.String())
	})

	n.Meow()
}

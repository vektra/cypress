package tools

import (
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
		var out cypress.ByteBuffer

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
		var out cypress.ByteBuffer

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
		var (
			out cypress.ByteBuffer
			exp cypress.ByteBuffer
		)

		m := cypress.Log()
		m.Add("hello", "world")

		gen := cypress.StaticGenerator(m)

		cat, err := NewCat(&out, gen, NATIVE)
		require.NoError(t, err)

		err = cat.Run()
		require.NoError(t, err)

		err = cypress.NewStreamEncoder(&exp).Receive(m)
		require.NoError(t, err)

		assert.Equal(t, exp.String(), out.String())
	})

	n.Meow()
}

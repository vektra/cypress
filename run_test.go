package cypress

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vektra/neko"
)

func TestRun(t *testing.T) {
	n := neko.Start(t)

	n.It("captures a processes stdout and turns outputs a message stream", func() {
		r, err := NewRun("echo", "hello there")
		require.NoError(t, err)

		m, err := r.Generate()
		require.NoError(t, err)

		cmd, ok := m.GetString("command")
		require.True(t, ok)

		assert.Equal(t, "echo", cmd)

		str, ok := m.GetString("message")
		require.True(t, ok)

		assert.Equal(t, "hello there", str)
	})

	n.Meow()
}

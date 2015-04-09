package json

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vektra/cypress"
	"github.com/vektra/neko"
)

func TestJson(t *testing.T) {
	n := neko.Start(t)

	n.It("expands a field that contains json", func() {
		m := cypress.Log()
		m.Add("message", `{"host": "zero", "status": "301"}`)

		j := &Filter{Field: "message"}

		m2, err := j.Filter(m)
		require.NoError(t, err)

		host, ok := m2.GetString("host")
		require.True(t, ok)

		assert.Equal(t, "zero", host)

		status, ok := m2.GetString("status")
		require.True(t, ok)

		assert.Equal(t, "301", status)
	})

	n.It("removes a field that has been expanded", func() {
		m := cypress.Log()
		m.Add("message", `{"host": "zero", "status": "301"}`)

		j := &Filter{Field: "message"}

		m2, err := j.Filter(m)
		require.NoError(t, err)

		_, ok := m2.GetString("message")
		assert.False(t, ok)
	})

	n.It("keys the expanded field if directed", func() {
		m := cypress.Log()
		m.Add("message", `{"host": "zero", "status": "301"}`)

		j := &Filter{Field: "message", Keep: true}

		m2, err := j.Filter(m)
		require.NoError(t, err)

		_, ok := m2.GetString("message")
		assert.True(t, ok)
	})

	n.Meow()
}

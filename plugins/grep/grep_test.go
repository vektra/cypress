package grep

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vektra/cypress"
	"github.com/vektra/neko"
)

func TestGrep(t *testing.T) {
	n := neko.Start(t)

	var mr cypress.MockReceiver

	n.CheckMock(&mr.Mock)

	n.It("filters messages that match the pattern through", func() {
		grep := &Grep{Field: "message", Pattern: "wo"}

		m := cypress.Log()
		m.Add("message", "hello world")

		m2 := cypress.Log()
		m2.Add("message", "hello people")

		m3, err := grep.Filter(m)
		require.NoError(t, err)

		assert.Equal(t, m, m3)

		m4, err := grep.Filter(m2)
		require.NoError(t, err)

		assert.Nil(t, m4)
	})

	n.It("can match a numeric field", func() {
		grep := &Grep{Field: "age", Pattern: "35"}

		m := cypress.Log()
		m.Add("age", 35)

		m2, err := grep.Filter(m)
		require.NoError(t, err)

		assert.Equal(t, m, m2)
	})

	n.Meow()
}

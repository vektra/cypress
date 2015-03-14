package cypress

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vektra/neko"
)

func TestKV(t *testing.T) {
	n := neko.Start(t)

	n.It("can parse out tags specified", func() {
		line := `> [region="us-west-1"] error="bad disks"` + "\n"

		m, err := ParseKV(line)
		require.NoError(t, err)

		assert.Equal(t, "region", m.Tags[0].Name)
		assert.Equal(t, "us-west-1", *m.Tags[0].Value)
	})

	n.It("can parse out tags multiple specified", func() {
		line := `> [region="us-west-1" host="betsy"] error="bad disks"` + "\n"

		m, err := ParseKV(line)
		require.NoError(t, err)

		assert.Equal(t, "region", m.Tags[0].Name)
		assert.Equal(t, "us-west-1", *m.Tags[0].Value)

		assert.Equal(t, "host", m.Tags[1].Name)
		assert.Equal(t, "betsy", *m.Tags[1].Value)
	})

	n.It("can parse out tags with ident values", func() {
		line := `> [region="us-west-1" host=betsy] error="bad disks"` + "\n"

		m, err := ParseKV(line)
		require.NoError(t, err)

		assert.Equal(t, "region", m.Tags[0].Name)
		assert.Equal(t, "us-west-1", *m.Tags[0].Value)

		assert.Equal(t, "host", m.Tags[1].Name)
		assert.Equal(t, "betsy", *m.Tags[1].Value)
	})

	n.It("can parse out value-less tags", func() {
		line := `> [region="us-west-1" !secure] error="bad disks"` + "\n"

		m, err := ParseKV(line)
		require.NoError(t, err)

		assert.Equal(t, "region", m.Tags[0].Name)
		assert.Equal(t, "us-west-1", *m.Tags[0].Value)

		assert.Equal(t, "secure", m.Tags[1].Name)
		assert.Nil(t, m.Tags[1].Value)
	})

	n.Meow()
}

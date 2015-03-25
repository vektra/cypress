package cypress

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vektra/neko"
)

func TestPlugins(t *testing.T) {
	n := neko.Start(t)

	n.It("creates a new instance of a requested plugin", func() {
		t1, ok := FindPlugin("Test")
		require.True(t, ok)

		t2, ok := FindPlugin("Test")
		require.True(t, ok)

		assert.True(t, reflect.ValueOf(t1).Pointer() != reflect.ValueOf(t2).Pointer())
	})

	n.It("searches for the plugins in a case insensitive way", func() {
		_, ok := FindPlugin("test")
		require.True(t, ok)

		_, ok = FindPlugin("tEST")
		require.True(t, ok)
	})

	n.Meow()
}

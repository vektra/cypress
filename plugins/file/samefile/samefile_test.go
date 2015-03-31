package samefile

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vektra/neko"
)

func TestSameFile(t *testing.T) {
	n := neko.Start(t)

	n.It("returns true when checked against a trivially same file", func() {
		tmpfile, err := ioutil.TempFile("", "samefile")
		require.NoError(t, err)

		defer os.Remove(tmpfile.Name())

		path := tmpfile.Name()

		id, err := Calculate(path)
		require.NoError(t, err)

		assert.True(t, Check(id, path))

		tmpfile2, err := ioutil.TempFile("", "samefile")
		require.NoError(t, err)

		defer os.Remove(tmpfile2.Name())

		path2 := tmpfile2.Name()

		assert.False(t, Check(id, path2))
	})

	n.It("returns false if there is a new file at the same path", func() {
		tmpfile, err := ioutil.TempFile("", "samefile")
		require.NoError(t, err)

		path := tmpfile.Name()

		defer os.Remove(path)

		id, err := Calculate(path)
		require.NoError(t, err)

		assert.True(t, Check(id, path))

		os.Remove(path)

		f, err := os.Create(path)
		require.NoError(t, err)

		defer f.Close()

		assert.False(t, Check(id, path))
	})

	n.Meow()
}

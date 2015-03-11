package cypress

import (
	"bytes"
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vektra/neko"
)

func TestConfig(t *testing.T) {
	n := neko.Start(t)

	paths := GlobalConfigPaths

	n.Cleanup(func() {
		GlobalConfigPaths = paths
	})

	n.It("reads the config via toml", func() {
		var buf bytes.Buffer

		buf.WriteString("[s3]\nallow_unsigned = true\n")

		cfg, err := ParseConfig(&buf)
		require.NoError(t, err)

		assert.True(t, cfg.S3.AllowUnsigned)
	})

	n.It("loads config from config paths", func() {
		tmpfile, err := ioutil.TempFile("", "config")
		require.NoError(t, err)

		defer os.Remove(tmpfile.Name())

		tmpfile.Write([]byte("[s3]\nallow_unsigned = true\n"))

		tmpfile.Close()

		GlobalConfigPaths = []string{tmpfile.Name()}

		cfg := loadGlobalConfig()

		assert.True(t, cfg.S3.AllowUnsigned)
	})

	n.Meow()
}

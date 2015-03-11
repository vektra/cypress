package keystore

import (
	"crypto/ecdsa"
	"crypto/rand"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vektra/neko"
)

func TestDirectory(t *testing.T) {
	n := neko.Start(t)

	tmpdir, err := ioutil.TempDir("", "keystore")
	require.NoError(t, err)

	defer os.RemoveAll(tmpdir)

	keydir := filepath.Join(tmpdir, "keys")

	n.Setup(func() {
		os.MkdirAll(keydir, 0700)
	})

	n.Cleanup(func() {
		os.RemoveAll(keydir)
	})

	n.It("reads PEM encoded private keys out of the directory", func() {
		key, err := ecdsa.GenerateKey(Curve, rand.Reader)
		require.NoError(t, err)

		err = SavePrivatePEM(filepath.Join(keydir, "test.pem"), key)
		require.NoError(t, err)

		dir, err := NewDirectory(keydir)
		require.NoError(t, err)

		out, err := dir.Get(KeyId(&key.PublicKey))
		require.NoError(t, err)

		assert.Equal(t, &key.PublicKey, out)
	})

	n.It("reads PEM encoded public keys out of the directory", func() {
		key, err := ecdsa.GenerateKey(Curve, rand.Reader)
		require.NoError(t, err)

		err = SavePublicPEM(filepath.Join(keydir, "test.pem"), &key.PublicKey)
		require.NoError(t, err)

		dir, err := NewDirectory(keydir)
		require.NoError(t, err)

		out, err := dir.Get(KeyId(&key.PublicKey))
		require.NoError(t, err)

		assert.Equal(t, &key.PublicKey, out)
	})

	n.It("returns an error if any key has the wrong perms", func() {
		key, err := ecdsa.GenerateKey(Curve, rand.Reader)
		require.NoError(t, err)

		path := filepath.Join(keydir, "test.pem")

		err = SavePrivatePEM(path, key)
		require.NoError(t, err)

		err = os.Chmod(path, 0644)
		require.NoError(t, err)

		_, err = NewDirectory(keydir)
		require.Error(t, err)
	})

	n.It("can find keys referenced by file name", func() {
		key, err := ecdsa.GenerateKey(Curve, rand.Reader)
		require.NoError(t, err)

		err = SavePrivatePEM(filepath.Join(keydir, "test.pem"), key)
		require.NoError(t, err)

		dir, err := NewDirectory(keydir)
		require.NoError(t, err)

		out, err := dir.Get("test")
		require.NoError(t, err)

		assert.Equal(t, &key.PublicKey, out)
	})

	n.It("can find keys referenced by pem header name (private)", func() {
		key, err := ecdsa.GenerateKey(Curve, rand.Reader)
		require.NoError(t, err)

		err = SaveNamedPrivatePEM(filepath.Join(keydir, "test.pem"), "foo", key)
		require.NoError(t, err)

		dir, err := NewDirectory(keydir)
		require.NoError(t, err)

		out, err := dir.Get("foo")
		require.NoError(t, err)

		assert.Equal(t, &key.PublicKey, out)
	})

	n.It("can find keys referenced by pem header name (public)", func() {
		key, err := ecdsa.GenerateKey(Curve, rand.Reader)
		require.NoError(t, err)

		err = SaveNamedPublicPEM(filepath.Join(keydir, "test.pem"), "foo", &key.PublicKey)
		require.NoError(t, err)

		dir, err := NewDirectory(keydir)
		require.NoError(t, err)

		out, err := dir.Get("foo")
		require.NoError(t, err)

		assert.Equal(t, &key.PublicKey, out)
	})

	n.Meow()
}

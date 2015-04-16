package file

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vektra/cypress/plugins/file/samefile"
	"github.com/vektra/neko"
)

func TestOffsetDB(t *testing.T) {
	n := neko.Start(t)

	tmpdir, err := ioutil.TempDir("", "offsetdb")
	require.NoError(t, err)

	defer os.RemoveAll(tmpdir)

	n.It("sets offset information", func() {
		fp := filepath.Join(tmpdir, "blah")

		defer os.Remove(fp)

		f, err := os.Create(fp)
		require.NoError(t, err)

		defer f.Close()

		db, err := NewOffsetDB(filepath.Join(tmpdir, "db"))
		require.NoError(t, err)

		err = db.Set(fp, 5)
		require.NoError(t, err)

		sum := sha256.Sum256([]byte(cleanPath(fp)))

		hash := hex.EncodeToString(sum[:])

		entryPath := filepath.Join(tmpdir, "db", hash[:2], hash)

		input, err := os.Open(entryPath)
		require.NoError(t, err)

		defer input.Close()

		var entry Entry

		err = json.NewDecoder(input).Decode(&entry)
		require.NoError(t, err)

		sfid, err := samefile.Calculate(cleanPath(fp))

		assert.Equal(t, cleanPath(fp), entry.Path)
		assert.Equal(t, int64(5), entry.Offset)
		assert.Equal(t, sfid, entry.SameFileID)
	})

	n.It("gets offset information", func() {
		fp := filepath.Join(tmpdir, "blah")

		defer os.Remove(fp)

		f, err := os.Create(fp)
		require.NoError(t, err)

		defer f.Close()

		db, err := NewOffsetDB(filepath.Join(tmpdir, "db"))
		require.NoError(t, err)

		err = db.Set(fp, 5)
		require.NoError(t, err)

		entry, err := db.Get(fp)
		require.NoError(t, err)

		assert.Equal(t, int64(5), entry.Offset)
	})

	n.It("validates true if the file is the same", func() {
		fp := filepath.Join(tmpdir, "blah")

		defer os.Remove(fp)

		f, err := os.Create(fp)
		require.NoError(t, err)

		defer f.Close()

		db, err := NewOffsetDB(filepath.Join(tmpdir, "db"))
		require.NoError(t, err)

		err = db.Set(fp, 0)
		require.NoError(t, err)

		entry, err := db.Get(fp)
		require.NoError(t, err)

		assert.True(t, entry.Valid())
	})

	n.It("validates false if the file is different", func() {
		fp := filepath.Join(tmpdir, "blah")

		defer os.Remove(fp)

		f, err := os.Create(fp)
		require.NoError(t, err)

		defer f.Close()

		db, err := NewOffsetDB(filepath.Join(tmpdir, "db"))
		require.NoError(t, err)

		err = db.Set(fp, 0)
		require.NoError(t, err)

		f.Close()

		f, err = os.Create(fp + ".new")
		require.NoError(t, err)

		os.Remove(fp)
		os.Rename(fp+".new", fp)

		defer f.Close()

		entry, err := db.Get(fp)
		require.NoError(t, err)

		assert.False(t, entry.Valid())
	})

	n.It("validates false if the file is smaller than the offset", func() {
		fp := filepath.Join(tmpdir, "blah")

		defer os.Remove(fp)

		f, err := os.Create(fp)
		require.NoError(t, err)

		defer f.Close()

		db, err := NewOffsetDB(filepath.Join(tmpdir, "db"))
		require.NoError(t, err)

		err = db.Set(fp, 5)
		require.NoError(t, err)

		f.Close()

		f, err = os.Create(fp)
		require.NoError(t, err)

		defer f.Close()

		entry, err := db.Get(fp)
		require.NoError(t, err)

		assert.False(t, entry.Valid())
	})

	n.Meow()
}

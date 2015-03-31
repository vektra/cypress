package file

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/vektra/cypress/plugins/file/samefile"
)

type OffsetDB struct {
	path string
}

func NewOffsetDB(path string) (*OffsetDB, error) {
	return &OffsetDB{path: path}, nil
}

type Entry struct {
	Path       string      `json:"path"`
	Offset     int64       `json:"offset"`
	SameFileID samefile.ID `json:"samefileid"`
}

func cleanPath(path string) string {
	one := filepath.Clean(path)
	two, err := filepath.EvalSymlinks(one)
	if err != nil {
		return one
	}

	three, err := filepath.Abs(two)
	if err != nil {
		return two
	}

	return three
}

func (o *OffsetDB) Set(path string, offset int64) error {
	path = cleanPath(path)

	sum := sha256.Sum256([]byte(path))

	hash := hex.EncodeToString(sum[:])

	dir := filepath.Join(o.path, hash[:2])

	err := os.MkdirAll(dir, 0755)
	if err != nil {
		return err
	}

	entryPath := filepath.Join(dir, hash)

	sfid, err := samefile.Calculate(path)
	if err != nil {
		return err
	}

	f, err := os.Create(entryPath)
	if err != nil {
		return err
	}

	defer f.Close()

	entry := &Entry{Path: path, Offset: offset, SameFileID: sfid}

	return json.NewEncoder(f).Encode(entry)
}

func (o *OffsetDB) Get(path string) (*Entry, error) {
	path = cleanPath(path)

	sum := sha256.Sum256([]byte(path))

	hash := hex.EncodeToString(sum[:])

	entryPath := filepath.Join(o.path, hash[:2], hash)

	input, err := os.Open(entryPath)
	if err != nil {
		if _, ok := err.(*os.PathError); ok {
			return nil, nil
		}

		return nil, err
	}

	var entry Entry

	err = json.NewDecoder(input).Decode(&entry)
	if err != nil {
		return nil, err
	}

	return &entry, nil
}

func (e *Entry) Valid() bool {
	if !samefile.Check(e.SameFileID, e.Path) {
		return false
	}

	stat, err := os.Stat(e.Path)
	if err != nil {
		return false
	}

	return stat.Size() >= e.Offset
}

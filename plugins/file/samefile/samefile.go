// Provides a file identifier that can be marshaled and checked
// later against a path

package samefile

import (
	"crypto/sha1"
	"encoding/hex"
)

type ID string

func Calculate(path string) (ID, error) {
	h := sha1.New()
	h.Write([]byte(path))

	err := fsHash(path, h)
	if err != nil {
		return "", err
	}

	return ID(hex.EncodeToString(h.Sum(nil))), nil
}

func Check(id ID, path string) bool {
	other, err := Calculate(path)
	if err != nil {
		return false
	}

	return id == other
}

package keystore

import (
	"crypto/ecdsa"
	"errors"
	"io/ioutil"
	"os"
	"path/filepath"
)

type Directory struct {
	dir string

	publicKeys  map[string]*ecdsa.PublicKey
	privateKeys map[string]*ecdsa.PrivateKey
}

var ErrNotDirectory = errors.New("path is not a directory")

func NewDirectory(path string) (*Directory, error) {
	s, err := os.Stat(path)
	if err != nil {
		return nil, err
	}

	if !s.IsDir() {
		return nil, ErrNotDirectory
	}

	dir := &Directory{
		dir:         path,
		publicKeys:  make(map[string]*ecdsa.PublicKey),
		privateKeys: make(map[string]*ecdsa.PrivateKey),
	}

	err = dir.loadKeys()
	if err != nil {
		return nil, err
	}

	return dir, nil
}

const (
	PEMPrivateKey = "EC PRIVATE KEY"
	PEMPublicKey  = "EC PUBLIC KEY"
)

func (d *Directory) loadKeys() error {
	ents, err := ioutil.ReadDir(d.dir)
	if err != nil {
		return err
	}

	for _, ent := range ents {
		val, err := LoadPEM(filepath.Join(d.dir, ent.Name()))
		if err != nil {
			return err
		}

		switch key := val.(type) {
		case *ecdsa.PrivateKey:
			keyid := KeyId(&key.PublicKey)

			d.publicKeys[keyid] = &key.PublicKey
			d.privateKeys[keyid] = key
		case *ecdsa.PublicKey:
			keyid := KeyId(key)

			d.publicKeys[keyid] = key
		default:
			return ErrUnknownKeyType
		}
	}

	return nil
}

func (d *Directory) Get(id string) (*ecdsa.PublicKey, error) {
	if key, ok := d.publicKeys[id]; ok {
		return key, nil
	}

	return nil, ErrUnknownKey
}

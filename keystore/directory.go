package keystore

import (
	"crypto/ecdsa"
	"errors"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
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
		val, blk, err := LoadPEM(filepath.Join(d.dir, ent.Name()))
		if err != nil {
			return err
		}

		alias := filepath.Base(ent.Name())

		if strings.HasSuffix(alias, ".pem") {
			alias = alias[:len(alias)-4]
		}

		switch key := val.(type) {
		case *ecdsa.PrivateKey:
			keyid := KeyId(&key.PublicKey)

			d.publicKeys[keyid] = &key.PublicKey
			d.privateKeys[keyid] = key
			d.publicKeys[alias] = &key.PublicKey
			d.privateKeys[alias] = key

			if cname, ok := blk.Headers[NameHeader]; ok {
				d.publicKeys[cname] = &key.PublicKey
				d.privateKeys[cname] = key
			}

		case *ecdsa.PublicKey:
			keyid := KeyId(key)

			d.publicKeys[keyid] = key
			d.publicKeys[alias] = key

			if cname, ok := blk.Headers[NameHeader]; ok {
				d.publicKeys[cname] = key
			}
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

func (d *Directory) GetPrivate(id string) (*ecdsa.PrivateKey, error) {
	if key, ok := d.privateKeys[id]; ok {
		return key, nil
	}

	return nil, ErrUnknownKey
}

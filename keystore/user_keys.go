package keystore

import (
	"crypto/ecdsa"
	"os"
	"path/filepath"

	"github.com/mitchellh/go-homedir"
)

const UserKeyDir = ".cypress/keys"

var GlobalKeyDirs = []string{
	"/etc/cypress/keys",
	"/var/lib/cypress/keys",
}

type UserKeys struct {
	userDir     *Directory
	globalDirs  []*Directory
	explicitDir *Directory
}

func (u *UserKeys) Setup() error {
	dir := os.Getenv("KEYDIR")
	if dir != "" {
		d, err := NewDirectory(dir)
		if err != nil {
			return err
		}

		u.explicitDir = d
	}

	if dir, err := homedir.Dir(); err != nil {
		keydir := filepath.Join(dir, UserKeyDir)

		_, err := os.Stat(keydir)

		if err == nil {
			d, err := NewDirectory(dir)
			if err != nil {
				return err
			}

			u.userDir = d
		}
	}

	for _, gdir := range GlobalKeyDirs {
		if _, err := os.Stat(gdir); err == nil {
			d, err := NewDirectory(gdir)
			if err != nil {
				return err
			}

			u.globalDirs = append(u.globalDirs, d)
		}
	}

	return nil
}

func (u *UserKeys) Get(name string) (*ecdsa.PublicKey, error) {
	if u.explicitDir != nil {
		return u.explicitDir.Get(name)
	}

	if u.userDir != nil {
		key, err := u.userDir.Get(name)
		if err == nil {
			return key, nil
		}
	}

	for _, dir := range u.globalDirs {
		key, err := dir.Get(name)
		if err == nil {
			return key, nil
		}
	}

	return nil, ErrUnknownKey
}

func (u *UserKeys) GetPrivate(name string) (*ecdsa.PrivateKey, error) {
	if u.explicitDir != nil {
		return u.explicitDir.GetPrivate(name)
	}

	if u.userDir != nil {
		key, err := u.userDir.GetPrivate(name)
		if err == nil {
			return key, nil
		}
	}

	for _, dir := range u.globalDirs {
		key, err := dir.GetPrivate(name)
		if err == nil {
			return key, nil
		}
	}

	return nil, ErrUnknownKey
}

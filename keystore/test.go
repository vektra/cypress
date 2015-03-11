package keystore

import (
	"crypto/ecdsa"
	"crypto/rand"
)
import "crypto/elliptic"

type TestKeys struct {
	Name string
	Key  *ecdsa.PrivateKey
}

func (t *TestKeys) Gen() *ecdsa.PrivateKey {
	key, err := ecdsa.GenerateKey(elliptic.P521(), rand.Reader)
	if err != nil {
		panic(err)
	}

	t.Key = key

	return key
}

func (t *TestKeys) Get(name string) (*ecdsa.PublicKey, error) {
	if t.Name != "" && t.Name != name {
		return nil, ErrUnknownKey
	}

	return &t.Key.PublicKey, nil
}

func (t *TestKeys) GetPrivate(name string) (*ecdsa.PrivateKey, error) {
	if t.Name != "" && t.Name != name {
		return nil, ErrUnknownKey
	}

	return t.Key, nil
}

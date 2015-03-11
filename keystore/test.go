package keystore

import (
	"crypto/ecdsa"
	"crypto/rand"
)
import "crypto/elliptic"

type TestKeys struct {
	Key *ecdsa.PrivateKey
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
	return &t.Key.PublicKey, nil
}

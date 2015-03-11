package keystore

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
)

func GenerateKey(path, name string) error {
	key, err := ecdsa.GenerateKey(elliptic.P521(), rand.Reader)
	if err != nil {
		return err
	}

	return SaveNamedPrivatePEM(path, name, key)
}

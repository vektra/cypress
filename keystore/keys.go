package keystore

import (
	"crypto/ecdsa"
)

type Keys interface {
	Get(name string) (*ecdsa.PublicKey, error)
	GetPrivate(name string) (*ecdsa.PrivateKey, error)
}

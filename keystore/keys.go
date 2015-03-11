package keystore

import (
	"crypto/ecdsa"
)

type Keys interface {
	Get(name string) (*ecdsa.PublicKey, error)
}

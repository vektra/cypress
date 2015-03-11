package keystore

import (
	"crypto/ecdsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
)

func KeyId(key *ecdsa.PublicKey) string {
	data, err := x509.MarshalPKIXPublicKey(key)
	if err != nil {
		panic(err)
	}

	sum := sha256.Sum256(data)
	return base64.StdEncoding.EncodeToString(sum[:])
}

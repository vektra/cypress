package keystore

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"io/ioutil"
	"os"
)

var ErrInvalidPermissions = errors.New("key has too wide of permissions")

func LoadPEM(path string) (interface{}, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	blk, _ := pem.Decode(data)

	switch blk.Type {
	case PEMPrivateKey:
		stat, err := os.Stat(path)
		if err != nil {
			return nil, err
		}

		if stat.Mode().Perm()&0077 != 0 {
			return nil, ErrInvalidPermissions
		}

		key, err := x509.ParseECPrivateKey(blk.Bytes)
		if err != nil {
			return nil, err
		}

		return key, nil
	case PEMPublicKey:
		x, y := elliptic.Unmarshal(Curve, blk.Bytes)

		key := &ecdsa.PublicKey{
			Curve: Curve,
			X:     x,
			Y:     y,
		}

		return key, nil
	default:
		return nil, ErrUnknownKeyType
	}
}

func SavePrivatePEM(path string, key *ecdsa.PrivateKey) error {
	bytes, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		return err
	}

	pb := &pem.Block{
		Type:  PEMPrivateKey,
		Bytes: bytes,
	}

	f, err := os.OpenFile(path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}

	err = pem.Encode(f, pb)
	if err != nil {
		return err
	}

	return f.Close()
}

func SavePublicPEM(path string, key *ecdsa.PublicKey) error {
	bytes := elliptic.Marshal(key.Curve, key.X, key.Y)

	pb := &pem.Block{
		Type:  PEMPublicKey,
		Bytes: bytes,
	}

	f, err := os.Create(path)
	if err != nil {
		return err
	}

	err = pem.Encode(f, pb)
	if err != nil {
		return err
	}

	return f.Close()
}

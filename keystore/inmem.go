package keystore

import "crypto/ecdsa"

type memoryKeys map[string]*ecdsa.PublicKey

func (m memoryKeys) Get(name string) (*ecdsa.PublicKey, error) {
	if k, ok := m[name]; ok {
		return k, nil
	}

	return nil, ErrUnknownKey
}

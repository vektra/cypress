package keystore

import "errors"

var (
	ErrUnknownKey     = errors.New("unknown key")
	ErrUnknownKeyType = errors.New("unknown key type")
)

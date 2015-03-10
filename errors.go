package cypress

import "errors"

var (
	ErrInvalidMessage    = errors.New("invalid message")
	ErrUnknownStreamType = errors.New("unknown stream type")
)

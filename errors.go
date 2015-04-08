package cypress

import "errors"

var (
	// The given message was invalid
	ErrInvalidMessage = errors.New("invalid message")

	// The system could not deduce the encoding of a stream
	ErrUnknownStreamType = errors.New("unknown stream type")
)

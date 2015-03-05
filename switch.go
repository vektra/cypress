package cypress

import (
	"errors"
	"io"
)

type Parser interface {
	Parse() error
}

type SwitchStream struct {
	Input  io.Reader
	Out    Reciever
	Source string

	parser Parser
}

var ErrUnknownStreamType = errors.New("unknown stream type")

type peekedInput struct {
	peek []byte
	rest io.Reader
}

func (pi *peekedInput) Read(data []byte) (int, error) {
	if pi.peek != nil {
		copy(data, pi.peek)
		n := len(pi.peek)
		pi.peek = nil
		return n, nil
	}

	return pi.rest.Read(data)
}

func (ss *SwitchStream) setup() (Parser, error) {
	data := make([]byte, 1)

	n, err := ss.Input.Read(data)
	if err != nil {
		return nil, err
	}

	if n != 1 {
		return nil, io.EOF
	}

	peeked := &peekedInput{data, ss.Input}

	switch data[0] {
	case '>':
		kv := &KVStream{
			Src:    peeked,
			Out:    ss.Out,
			Source: ss.Source,
		}

		return kv, nil
	case '+':
		pb := &PBStream{
			Src: peeked,
			Out: ss.Out,
		}

		return pb, nil
	case '{':
		js := &JsonStream{
			Src: peeked,
			Out: ss.Out,
		}

		return js, nil
	default:
		return nil, ErrUnknownStreamType
	}
}

func (ss *SwitchStream) Parse() error {
	if ss.parser == nil {
		parser, err := ss.setup()
		if err != nil {
			return err
		}

		ss.parser = parser
	}

	ss.parser.Parse()
	return nil
}

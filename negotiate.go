package cypress

import (
	"io"

	"compress/zlib"

	"code.google.com/p/snappy-go/snappy"
)

type StreamNegotiator struct {
	r   io.Reader
	out Receiver
}

func NewStreamNegotiator(r io.Reader, out Receiver) *StreamNegotiator {
	return &StreamNegotiator{r, out}
}

func (s *StreamNegotiator) Negotiate() (Parser, error) {
	data := make([]byte, 1)

	n, err := s.r.Read(data)
	if err != nil {
		return nil, err
	}

	if n != 1 {
		return nil, io.EOF
	}

	if data[0] == '-' {
		err = s.processHeader(data)
		if err != nil {
			return nil, err
		}

	} else {
		s.r = &peekedInput{data, s.r}
	}

	return &SwitchStream{Input: s.r, Out: s.out}, nil
}

func (s *StreamNegotiator) processHeader(buf []byte) error {
	cnt, err := ReadUvarint(s.r, buf)
	if err != nil {
		return err
	}

	data := make([]byte, cnt)

	_, err = io.ReadFull(s.r, data)
	if err != nil {
		return err
	}

	hdr := &StreamHeader{}
	err = hdr.Unmarshal(data)
	if err != nil {
		return err
	}

	switch hdr.GetCompression() {
	case StreamHeader_SNAPPY:
		s.r = snappy.NewReader(s.r)
	case StreamHeader_ZLIB:
		zr, err := zlib.NewReader(s.r)
		if err != nil {
			return err
		}

		s.r = zr
	}

	return nil
}

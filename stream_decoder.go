package cypress

import "io"

type StreamDecoder struct {
	r    io.Reader
	init bool
	dec  *Decoder

	Header *StreamHeader
}

func NewStreamDecoder(r io.Reader) (*StreamDecoder, error) {
	return &StreamDecoder{r: r, dec: NewDecoder(r)}, nil
}

func (s *StreamDecoder) Probe() error {
	s.init = true

	probe := NewProbe(s.r)

	err := probe.Probe()
	if err != nil {
		return err
	}

	s.Header = probe.Header

	s.dec = NewDecoder(probe.Reader())

	return nil
}

func (s *StreamDecoder) Generate() (*Message, error) {
	if !s.init {
		err := s.Probe()
		if err != nil {
			return nil, err
		}
	}

	return s.dec.Decode()
}

func (s *StreamDecoder) Close() error {
	return nil
}

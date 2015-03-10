package cypress

import "io"

type StreamDecoder struct {
	r   io.Reader
	dec *Decoder
}

func NewStreamDecoder(r io.Reader) *StreamDecoder {
	return &StreamDecoder{r: r, dec: NewDecoder(r)}
}

func (s *StreamDecoder) Init() error {
	probe := NewProbe(s.r)

	err := probe.Probe()
	if err != nil {
		return err
	}

	s.dec = NewDecoder(probe.Reader())

	return nil
}

func (s *StreamDecoder) Generate() (*Message, error) {
	return s.dec.Decode()
}

func (s *StreamDecoder) Close() error {
	return nil
}

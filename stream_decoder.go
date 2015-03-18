package cypress

import "io"

type StreamDecoder struct {
	r   io.Reader
	dec *Decoder

	Header *StreamHeader
}

func NewStreamDecoder(r io.Reader) (*StreamDecoder, error) {
	sd := &StreamDecoder{r: r, dec: NewDecoder(r)}

	err := sd.init()
	if err != nil {
		return nil, err
	}

	return sd, nil
}

func (s *StreamDecoder) init() error {
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
	return s.dec.Decode()
}

func (s *StreamDecoder) Close() error {
	return nil
}

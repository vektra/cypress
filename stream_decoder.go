package cypress

import "io"

// A type which uses Probe and Decoder generate Messages
type StreamDecoder struct {
	r    io.Reader
	init bool
	dec  *Decoder

	Header *StreamHeader
}

// Create a new StreamDecoder from the data in r
func NewStreamDecoder(r io.Reader) (*StreamDecoder, error) {
	return &StreamDecoder{r: r, dec: NewDecoder(r)}, nil
}

// Probe the stream and setup the decoder to read Messages
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

// Read the next Message in the stream. If the stream has not
// been initialized, Probe() is called first.
func (s *StreamDecoder) Generate() (*Message, error) {
	if !s.init {
		err := s.Probe()
		if err != nil {
			return nil, err
		}
	}

	return s.dec.Decode()
}

// To satisify the Generator interface
func (s *StreamDecoder) Close() error {
	return nil
}

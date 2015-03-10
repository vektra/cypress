package cypress

import (
	"io"
	"os"
)

type StreamEncoder struct {
	w       io.Writer
	enc     *Encoder
	encoded uint64
}

func NewStreamEncoder(w io.Writer) *StreamEncoder {
	return &StreamEncoder{w: w, enc: NewEncoder(w)}
}

var StreamNotifyByte = []byte{'-'}

func (s *StreamEncoder) WriteHeader(comp StreamHeader_Compression) error {
	_, err := s.w.Write(StreamNotifyByte)
	if err != nil {
		return err
	}

	hdr := &StreamHeader{Compression: comp.Enum()}

	data, err := hdr.Marshal()
	if err != nil {
		return err
	}

	_, err = WriteUvarint(s.w, uint64(len(data)))
	if err != nil {
		return err
	}

	_, err = s.w.Write(data)
	return err
}

func (s *StreamEncoder) Init(comp StreamHeader_Compression) error {
	err := s.WriteHeader(comp)
	if err != nil {
		return err
	}

	s.enc = NewEncoder(WriteCompressed(s.w, comp))

	return nil
}

func (s *StreamEncoder) OpenFile(f *os.File) error {
	probe := NewProbe(f)

	err := probe.Probe()
	if err != nil {
		return err
	}

	s.enc = NewEncoder(WriteCompressed(s.w, probe.Compression()))

	_, err = f.Seek(0, os.SEEK_END)
	return err
}

func (s *StreamEncoder) Receive(m *Message) error {
	cnt, err := s.enc.Encode(m)

	s.encoded += cnt

	return err
}

func (s *StreamEncoder) EncodedBytes() uint64 {
	return s.encoded
}

package cypress

import (
	"io"
	"os"

	"github.com/andrew-d/go-termutil"
)

type MessageEncoder interface {
	Encode(m *Message) (uint64, error)
}

// A type that encodes Messages to a stream with optional compression
type StreamEncoder struct {
	w       io.Writer
	enc     MessageEncoder
	encoded uint64
}

// Create a new StreamEncoder sending data to w
func NewStreamEncoder(w io.Writer) *StreamEncoder {
	if w == os.Stdout {
		if termutil.Isatty(os.Stdout.Fd()) {
			return &StreamEncoder{w: w, enc: NewKVEncoder(w)}
		}
	}

	return &StreamEncoder{w: w, enc: NewEncoder(w)}
}

var StreamNotifyByte = []byte{'-'}

// Initialize the StreamEncoder to a particular compression level and
// write the header
func (s *StreamEncoder) Init(comp StreamHeader_Compression) error {
	hdr := &StreamHeader{Compression: comp.Enum()}

	return s.WriteCustomHeader(hdr)
}

// Write a StreamHeader
func (s *StreamEncoder) WriteCustomHeader(hdr *StreamHeader) error {
	_, err := s.w.Write(StreamNotifyByte)
	if err != nil {
		return err
	}

	data, err := hdr.Marshal()
	if err != nil {
		return err
	}

	_, err = WriteUvarint(s.w, uint64(len(data)))
	if err != nil {
		return err
	}

	_, err = s.w.Write(data)
	if err != nil {
		return err
	}

	s.enc = NewEncoder(WriteCompressed(s.w, hdr.GetCompression()))

	return nil
}

// Probe the file and setup the encoder to match the probe's
// settings.
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

// Take a Message and encode it
func (s *StreamEncoder) Receive(m *Message) error {
	cnt, err := s.enc.Encode(m)

	s.encoded += cnt

	return err
}

// Indicate how many bytes have been sent
func (s *StreamEncoder) EncodedBytes() uint64 {
	return s.encoded
}

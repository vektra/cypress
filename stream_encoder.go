package cypress

import (
	"io"
	"os"
	"sync"
	"time"

	"github.com/andrew-d/go-termutil"
	"gopkg.in/tomb.v2"
)

type MessageEncoder interface {
	Encode(m *Message) (uint64, error)
}

type Flusher interface {
	Flush() error
}

// A type that encodes Messages to a stream with optional compression
type StreamEncoder struct {
	w       io.WriteCloser
	ew      io.WriteCloser
	flush   Flusher
	enc     MessageEncoder
	encoded uint64
	lock    sync.Mutex

	t tomb.Tomb
}

// Create a new StreamEncoder sending data to w
func NewStreamEncoder(w io.WriteCloser) *StreamEncoder {
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

func (s *StreamEncoder) flushTimer() error {
	tick := time.NewTicker(5 * time.Second)

	for {
		select {
		case <-tick.C:
			s.lock.Lock()
			s.flush.Flush()
			s.lock.Unlock()
		case <-s.t.Dying():
			return nil
		}
	}
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

	s.ew = WriteCompressed(s.w, hdr.GetCompression())

	if f, ok := s.ew.(Flusher); ok {
		s.flush = f
		s.t.Go(s.flushTimer)
	}

	s.enc = NewEncoder(s.ew)

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

	s.ew = WriteCompressed(s.w, probe.Compression())

	if f, ok := s.ew.(Flusher); ok {
		s.flush = f
		s.t.Go(s.flushTimer)
	}

	s.enc = NewEncoder(s.ew)

	_, err = f.Seek(0, os.SEEK_END)
	return err
}

// Take a Message and encode it
func (s *StreamEncoder) Receive(m *Message) error {
	s.lock.Lock()

	cnt, err := s.enc.Encode(m)

	s.encoded += cnt

	s.lock.Unlock()

	return err
}

func (s *StreamEncoder) Close() error {
	s.t.Kill(nil)
	s.t.Wait()
	return s.ew.Close()
}

func (s *StreamEncoder) Flush() error {
	if s.flush == nil {
		return nil
	}

	s.lock.Lock()
	defer s.lock.Unlock()

	return s.flush.Flush()
}

// Indicate how many bytes have been sent
func (s *StreamEncoder) EncodedBytes() uint64 {
	return s.encoded
}

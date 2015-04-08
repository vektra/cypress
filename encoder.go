package cypress

import (
	"encoding/binary"
	"io"
)

// A type which encodes messages given to it in native protobuf format
// and writes them out.
type Encoder struct {
	w io.Writer
}

// Create an Encoder that will write it's output to w
func NewEncoder(w io.Writer) *Encoder {
	return &Encoder{
		w: w,
	}
}

// Encode and write a Message
func (e *Encoder) Encode(m *Message) (uint64, error) {
	sz := m.Size()

	buf := pbBufPool.Get().([]byte)

	buf[0] = '+'

	cnt := binary.PutUvarint(buf[1:], uint64(sz))

	_, err := e.w.Write(buf[:cnt+1])
	if err != nil {
		pbBufPool.Put(buf)
		return 0, err
	}

	if len(buf) < sz {
		buf = make([]byte, sz)
	}

	cnt, err = m.MarshalTo(buf)
	if err != nil {
		pbBufPool.Put(buf)
		return 0, err
	}

	_, err = e.w.Write(buf[:cnt])

	pbBufPool.Put(buf)

	if err != nil {
		return 0, err
	}

	return uint64(sz) + 5, nil
}

// An encoder that writes messages in Key/Value format
type KVEncoder struct {
	w io.Writer
}

// Create a KVEncoder that writes it's output to w
func NewKVEncoder(w io.Writer) *KVEncoder {
	return &KVEncoder{w}
}

// Encode and write a message
func (kv *KVEncoder) Encode(m *Message) (uint64, error) {
	str := m.KVString()

	kv.w.Write([]byte(str))
	kv.w.Write([]byte("\n"))

	return uint64(len(str) + 1), nil
}

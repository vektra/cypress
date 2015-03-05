package cypress

import (
	"encoding/binary"
	"errors"
	"io"

	"github.com/gogo/protobuf/proto"
)

var ErrBadStream = errors.New("bad stream detected")

func (m *Message) ReadWire(c io.Reader) (int, error) {
	var buf [4]byte

	_, err := io.ReadFull(c, buf[:])

	if err != nil {
		return 0, err
	}

	sz := binary.BigEndian.Uint32(buf[:])

	mbuf := make([]byte, sz)

	_, err = io.ReadFull(c, mbuf)

	if err != nil {
		return 0, err
	}

	return int(sz) + 4, m.Unmarshal(mbuf)
}

func (m *Message) WriteWire(c io.Writer) (int, error) {
	data, err := m.Marshal()

	if err != nil {
		return 0, err
	}

	var buf [4]byte

	binary.BigEndian.PutUint32(buf[:], uint32(len(data)))

	n, err := c.Write(buf[:])

	if err != nil {
		return 0, err
	}

	if n != 4 {
		return 0, io.ErrShortWrite
	}

	n, err = c.Write(data)

	if err != nil {
		return 0, err
	}

	if n != len(data) {
		return 0, io.ErrShortWrite
	}

	return n + 4, nil
}

func FromProtobuf(buf []byte) (*Message, error) {
	m := &Message{}

	err := proto.Unmarshal(buf, m)
	if err != nil {
		return nil, err
	}

	return m, nil
}

func ToProtobuf(m *Message) ([]byte, error) {
	return m.Marshal()
}

type PBStream struct {
	Src io.Reader
	Out Reciever
}

func (pb *PBStream) Parse() error {
	tag := make([]byte, 1)

	for {
		_, err := pb.Src.Read(tag)
		if err != nil {
			return err
		}

		if tag[0] != '+' {
			return ErrBadStream
		}

		m := &Message{}
		_, err = m.ReadWire(pb.Src)
		if err != nil {
			return err
		}

		pb.Out.Read(m)
	}
}

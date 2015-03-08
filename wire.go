package cypress

import (
	"errors"
	"io"

	"github.com/gogo/protobuf/proto"
)

var ErrBadStream = errors.New("bad stream detected")

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
	Out Receiver
}

func (pb *PBStream) Parse() error {
	dec := NewDecoder()

	for {
		m, err := dec.DecodeFrom(pb.Src)
		if err != nil {
			return err
		}

		pb.Out.Read(m)
	}

	return nil
}

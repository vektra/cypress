package cypress

import (
	"encoding/binary"
	"io"
)

type WireMessage struct {
	msg  *Message
	data []byte
}

func (wm *WireMessage) Message() (*Message, error) {
	if wm.msg == nil {
		var msg Message
		err := msg.Unmarshal(wm.data)
		if err != nil {
			return nil, err
		}

		wm.msg = &msg
	}

	return wm.msg, nil
}

func (wm *WireMessage) Marshal() ([]byte, error) {
	if wm.data == nil {
		data, err := wm.msg.Marshal()
		if err != nil {
			return nil, err
		}

		wm.data = data
	}

	return wm.data, nil
}

func ReadWireMessage(c io.Reader) (*WireMessage, error) {
	var buf [4]byte

	_, err := io.ReadFull(c, buf[:])

	if err != nil {
		return nil, err
	}

	sz := binary.BigEndian.Uint32(buf[:])

	mbuf := make([]byte, sz)

	_, err = io.ReadFull(c, mbuf)

	if err != nil {
		return nil, err
	}

	return &WireMessage{data: mbuf}, nil
}

func WriteWireMessage(c io.Writer, m *WireMessage) (int, error) {
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

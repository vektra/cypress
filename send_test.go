package cypress

import (
	"bytes"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vektra/neko"
)

type dualBuffer struct {
	read  *bytes.Buffer
	write *bytes.Buffer
}

func newDualBuffer() *dualBuffer {
	return &dualBuffer{
		new(bytes.Buffer),
		new(bytes.Buffer),
	}
}

func (db *dualBuffer) Read(data []byte) (int, error) {
	return db.read.Read(data)
}

func (db *dualBuffer) Write(data []byte) (int, error) {
	return db.write.Write(data)
}

func (db *dualBuffer) Flip() *dualBuffer {
	return &dualBuffer{db.write, db.read}
}

func TestSend(t *testing.T) {
	n := neko.Start(t)

	n.It("sends a handshake header", func() {
		db := newDualBuffer()

		s := NewSend(db, 0)

		err := s.SendHandshake()
		require.NoError(t, err)

		var hdr StreamHeader

		c, err := db.write.ReadByte()
		require.NoError(t, err)

		assert.Equal(t, StreamNotifyByte[0], c)

		err = hdr.UnmarshalFrom(db.write)
		require.NoError(t, err)

		assert.Equal(t, StreamHeader_RELIABLE, hdr.GetMode())
	})

	n.It("writes a message", func() {
		db := newDualBuffer()

		s := NewSend(db, 0)

		m := Log()
		m.Add("hello", "world")

		err := s.transmit(m)
		require.NoError(t, err)

		dec := NewDecoder(db.write)

		m2, err := dec.Decode()
		require.NoError(t, err)

		assert.Equal(t, m, m2)
	})

	n.It("reads an ack from the remote side", func() {
		db := newDualBuffer()

		s := NewSend(db, 0)

		db.read.WriteString("k")

		err := s.readAck()
		require.NoError(t, err)
	})

	n.It("has readAck return an error if the stream is closed", func() {
		db := newDualBuffer()

		s := NewSend(db, 0)

		err := s.readAck()
		require.Equal(t, err, io.EOF)
	})

	n.It("has readAck return an error if the stream doesn't have an ack byte", func() {
		db := newDualBuffer()

		s := NewSend(db, 0)

		db.read.WriteString("c")

		err := s.readAck()
		require.Equal(t, err, ErrStreamUnsynced)
	})

	n.It("sends a message and waits for the ack", func() {
		db := newDualBuffer()

		s := NewSend(db, 0)

		m := Log()
		m.Add("hello", "world")

		db.read.WriteString("k")

		err := s.Receive(m)
		require.NoError(t, err)

		dec := NewDecoder(db.write)

		m2, err := dec.Decode()
		require.NoError(t, err)

		assert.Equal(t, m, m2)
	})

	n.It("only reads for acks when the available window slots is depleted", func() {
		db := newDualBuffer()

		s := NewSend(db, 2)

		m := Log()
		m.Add("hello", "world")

		err := s.Receive(m)
		require.NoError(t, err)

		assert.Equal(t, s.available, 1)

		err = s.Receive(m)
		require.NoError(t, err)

		assert.Equal(t, s.available, 0)

		db.read.WriteString("kkk")

		err = s.Receive(m)
		require.NoError(t, err)

		_, err = db.read.ReadByte()
		assert.Equal(t, err, io.EOF)
	})

	n.Meow()
}

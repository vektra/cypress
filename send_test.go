package cypress

import (
	"bytes"
	"io"
	"os"
	"syscall"
	"testing"
	"time"

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

type dualPipes struct {
	read  *io.PipeReader
	write *io.PipeWriter
}

func (db *dualPipes) Read(data []byte) (int, error) {
	return db.read.Read(data)
}

func (db *dualPipes) Write(data []byte) (int, error) {
	return db.write.Write(data)
}

func newDualPipes() (*dualPipes, *dualPipes) {
	ar, aw := io.Pipe()
	br, bw := io.Pipe()

	return &dualPipes{ar, bw}, &dualPipes{br, aw}
}

func newPair() (*os.File, *os.File) {
	fd, err := syscall.Socketpair(syscall.AF_UNIX, syscall.SOCK_STREAM, 0)
	if err != nil {
		panic(err)
	}

	return os.NewFile(uintptr(fd[0]), "sockepair-1"),
		os.NewFile(uintptr(fd[1]), "socketpair-2")
}

func TestSend(t *testing.T) {
	n := neko.Start(t)

	var ack MockSendRequest

	n.CheckMock(&ack.Mock)

	n.It("sends a handshake header", func() {
		db := newDualBuffer()

		s := NewSend(db, NoWindow)

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

		s := NewSend(db, NoWindow)

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

		s := NewSend(db, NoWindow)

		db.read.WriteString("k")

		err := s.readAck()
		require.NoError(t, err)
	})

	n.It("has readAck return an error if the stream is closed", func() {
		db := newDualBuffer()

		s := NewSend(db, NoWindow)

		err := s.readAck()
		require.Equal(t, err, io.EOF)
	})

	n.It("has readAck return an error if the stream doesn't have an ack byte", func() {
		db := newDualBuffer()

		s := NewSend(db, NoWindow)

		db.read.WriteString("c")

		err := s.readAck()
		require.Equal(t, err, ErrStreamUnsynced)
	})

	n.It("sends a message and waits for the ack", func() {
		db := newDualBuffer()

		s := NewSend(db, NoWindow)

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
		send, recv := newPair()
		defer send.Close()
		defer recv.Close()

		s := NewSend(send, 3)

		m := Log()
		m.Add("hello", "world")

		err := s.Receive(m)
		require.NoError(t, err)

		assert.Equal(t, s.available, int32(2))

		err = s.Receive(m)
		require.NoError(t, err)

		assert.Equal(t, s.available, int32(1))

		go func() {
			time.Sleep(1)
			recv.Write([]byte("kkk"))
		}()

		err = s.Receive(m)
		require.NoError(t, err)

		assert.Equal(t, s.available, int32(3))
	})

	n.It("can calculate minimize windows to use", func() {
		assert.Equal(t, 300, MinimumSendWindow(15*time.Millisecond, 1000))
	})

	n.It("sends acks for messages that are received", func() {
		db := newDualBuffer()

		s := NewSend(db, NoWindow)

		m := Log()
		m.Add("hello", "world")

		db.read.WriteString("k")

		ack.On("Ack", m).Return(nil)

		err := s.Send(m, &ack)
		require.NoError(t, err)

		dec := NewDecoder(db.write)

		m2, err := dec.Decode()
		require.NoError(t, err)

		assert.Equal(t, m, m2)
	})

	n.It("sends acks for proper messages", func() {
		db := newDualBuffer()

		s := NewSend(db, 0)

		m := Log()
		m.Add("hello", "world")

		m2 := Log()
		m2.Add("message", "logs are fun")

		ack.On("Ack", m).Return(nil)
		ack.On("Nack", m2).Return(nil)

		err := s.Send(m, &ack)
		require.NoError(t, err)

		err = s.Send(m2, &ack)
		require.NoError(t, err)

		db.read.WriteString("k")

		time.Sleep(100 * time.Millisecond)
	})

	n.It("sends nacks for messages inflight when an error is seen", func() {
		send, recv := newPair()
		defer send.Close()
		defer recv.Close()

		s := NewSend(send, 0)

		m := Log()
		m.Add("hello", "world")

		ack.On("Nack", m).Return(nil)

		err := s.Send(m, &ack)
		require.NoError(t, err)

		err = recv.Close()
		require.NoError(t, err)

		// let backgroundAck routine detect the error
		time.Sleep(100 * time.Millisecond)
	})

	n.It("sends nacks for messages inflight when an error is seen on transmit", func() {
		send, recv := newPair()
		defer send.Close()
		defer recv.Close()

		s := NewSend(send, 0)

		m := Log()
		m.Add("hello", "world")

		ack.On("Nack", m).Return(nil)

		err := recv.Close()
		require.NoError(t, err)

		time.Sleep(100 * time.Millisecond)

		assert.Equal(t, s.closed, true)
		s.closed = false

		err = s.Send(m, &ack)
		require.Equal(t, ErrClosed, err)
	})

	n.It("sends nacks for messages sent when already closed", func() {
		send, recv := newPair()
		defer send.Close()
		defer recv.Close()

		s := NewSend(send, 0)

		m := Log()
		m.Add("hello", "world")

		ack.On("Nack", m).Return(nil)

		s.closed = true

		err := s.Send(m, &ack)
		require.Equal(t, ErrClosed, err)
	})

	n.Meow()
}

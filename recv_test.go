package cypress

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vektra/neko"
)

func TestRecv(t *testing.T) {
	n := neko.Start(t)

	n.It("receives messages", func() {
		db := newDualBuffer()

		s := NewSend(db.Flip(), 0)

		err := s.SendHandshake()
		require.NoError(t, err)

		r, err := NewRecv(db)
		require.NoError(t, err)

		m := Log()
		m.Add("hello", "world")

		err = s.transmit(m)
		require.NoError(t, err)

		m2, err := r.recvMessage()
		require.NoError(t, err)

		assert.Equal(t, m, m2)
	})

	n.It("sends an ack back", func() {
		db := newDualBuffer()

		s := NewSend(db.Flip(), 0)

		err := s.SendHandshake()
		require.NoError(t, err)

		r, err := NewRecv(db)
		require.NoError(t, err)

		err = r.sendAck()
		require.NoError(t, err)

		err = s.readAck()
		require.NoError(t, err)
	})

	n.It("reads a message and sends an ack", func() {
		db := newDualBuffer()

		s := NewSend(db.Flip(), 0)

		err := s.SendHandshake()
		require.NoError(t, err)

		r, err := NewRecv(db)
		require.NoError(t, err)

		m := Log()
		m.Add("hello", "world")

		err = s.transmit(m)
		require.NoError(t, err)

		m2, err := r.Generate()
		require.NoError(t, err)

		assert.Equal(t, m, m2)

		err = s.readAck()
		require.NoError(t, err)
	})

	n.It("does not ack messages if the header didn't indicate reliable", func() {
		db := newDualBuffer()

		s := NewStreamEncoder(db.Flip())

		err := s.Init(NONE)
		require.NoError(t, err)

		r, err := NewRecv(db)
		require.NoError(t, err)

		m := Log()
		m.Add("hello", "world")

		err = s.Receive(m)
		require.NoError(t, err)

		m2, err := r.Generate()
		require.NoError(t, err)

		assert.Equal(t, m, m2)

		assert.Equal(t, 0, db.write.Len())
	})

	n.Meow()
}

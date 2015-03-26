package tcp

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vektra/cypress"
	"github.com/vektra/neko"
)

func TestTCPRecv(t *testing.T) {
	n := neko.Start(t)

	n.It("sets up generators for each new connection", func() {
		c := make(chan cypress.Generator)

		r, err := NewTCPRecv(":0", cypress.GeneratorHandlerFunc(func(g cypress.Generator) {
			c <- g
		}))

		require.NoError(t, err)

		err = r.Listen()
		require.NoError(t, err)

		go r.Accept()

		defer r.Close()

		addr := r.l.Addr().String()

		m := cypress.Log()
		m.Add("hello", "world")

		s, err := NewTCPSend([]string{addr}, 0, DefaultTCPBuffer)
		require.NoError(t, err)

		err = s.Receive(m)
		require.NoError(t, err)

		defer s.Close()

		gout := <-c

		m2, err := gout.Generate()
		require.NoError(t, err)

		assert.Equal(t, m, m2)

		s.Close()
	})

	n.It("has a generator form that combines all input generators", func() {
		gen, err := NewTCPRecvGenerator(":0")
		require.NoError(t, err)

		err = gen.Listen()
		require.NoError(t, err)

		go gen.Accept()

		defer gen.Close()

		addr := gen.l.Addr().String()

		m := cypress.Log()
		m.Add("hello", "world")

		go func() {
			s, err := NewTCPSend([]string{addr}, 0, DefaultTCPBuffer)
			require.NoError(t, err)

			err = s.Receive(m)
			require.NoError(t, err)

			s.Close()
		}()

		m2, err := gen.Generate()
		require.NoError(t, err)

		assert.Equal(t, m, m2)

	})

	n.Meow()
}

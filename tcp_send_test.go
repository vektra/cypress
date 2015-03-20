package cypress

import (
	"net"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vektra/neko"
)

func TestTCPSend(t *testing.T) {
	n := neko.Start(t)

	n.It("sends logs across to a server", func() {
		l, err := net.Listen("tcp", ":0")
		require.NoError(t, err)

		defer l.Close()

		var recvMesg *Message

		var wg sync.WaitGroup

		wg.Add(1)
		go func() {
			defer wg.Done()

			c, err := l.Accept()
			require.NoError(t, err)

			recv, err := NewRecv(c)

			recvMesg, err = recv.Generate()
		}()

		m := Log()
		m.Add("hello", "world")

		tcp, err := NewTCPSend(l.Addr().String(), 0, 0)
		require.NoError(t, err)

		defer tcp.Close()

		err = tcp.Receive(m)
		require.NoError(t, err)

		wg.Wait()

		assert.Equal(t, m, recvMesg)
	})

	n.It("uses a window to increase throughput", func() {
		time.Sleep(1 * time.Second)

		l, err := net.Listen("tcp", ":0")
		require.NoError(t, err)

		defer l.Close()

		var wg sync.WaitGroup

		latch := make(chan bool)

		wg.Add(1)
		go func() {
			defer wg.Done()

			c, err := l.Accept()
			require.NoError(t, err)

			defer c.Close()

			recv, err := NewRecv(c)
			require.NoError(t, err)

			<-latch

			_, err = recv.Generate()
			require.NoError(t, err)

			latch <- true
			<-latch
		}()

		m := Log()
		m.Add("hello", "world")

		tcp, err := NewTCPSend(l.Addr().String(), 0, 0)
		require.NoError(t, err)

		defer tcp.Close()

		err = tcp.Receive(m)
		require.NoError(t, err)

		err = tcp.Receive(m)
		require.NoError(t, err)

		runtime.Gosched()

		assert.Equal(t, 2, tcp.outstanding)

		latch <- true
		<-latch

		time.Sleep(100 * time.Millisecond)

		assert.Equal(t, 1, tcp.outstanding)

		latch <- true
		wg.Wait()

		runtime.Gosched()

		assert.Equal(t, 1, tcp.outstanding)
	})

	n.It("send nack'd messages once reconnected", func() {
		time.Sleep(1 * time.Second)

		l, err := net.Listen("tcp", ":0")
		require.NoError(t, err)

		defer l.Close()

		var recvMesg *Message
		var wg sync.WaitGroup

		latch := make(chan bool)

		wg.Add(1)
		go func() {
			defer wg.Done()

			c, err := l.Accept()
			require.NoError(t, err)

			defer c.Close()

			recv, err := NewRecv(c)
			require.NoError(t, err)

			<-latch

			_, err = recv.Generate()
			require.NoError(t, err)

			c.Close()

			c, err = l.Accept()
			require.NoError(t, err)

			defer c.Close()

			recv, err = NewRecv(c)
			require.NoError(t, err)

			recvMesg, err = recv.Generate()
			require.NoError(t, err)

			latch <- true
		}()

		m := Log()
		m.Add("hello", "world")

		tcp, err := NewTCPSend(l.Addr().String(), 0, 0)
		require.NoError(t, err)

		defer tcp.Close()

		err = tcp.Receive(m)
		require.NoError(t, err)

		m2 := Log()
		m2.Add("hello", "world")

		err = tcp.Receive(m2)
		require.NoError(t, err)

		latch <- true

		select {
		case <-time.Tick(100 * time.Millisecond):
			t.Fatal("didn't generate a new message")
		case <-latch:
			// ok
		}

		wg.Wait()

		assert.Equal(t, m2, recvMesg)
	})

	n.Meow()
}

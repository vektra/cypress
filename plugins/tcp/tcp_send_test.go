package tcp

import (
	"net"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vektra/cypress"
	"github.com/vektra/neko"
)

func TestTCPSend(t *testing.T) {
	n := neko.Start(t)

	n.It("sends logs across to a server", func() {
		l, err := net.Listen("tcp", ":0")
		require.NoError(t, err)

		defer l.Close()

		var recvMesg *cypress.Message

		var wg sync.WaitGroup

		wg.Add(1)
		go func() {
			defer wg.Done()

			c, err := l.Accept()
			require.NoError(t, err)

			recv, err := cypress.NewRecv(c)

			recvMesg, err = recv.Generate()
		}()

		m := cypress.Log()
		m.Add("hello", "world")

		tcp, err := NewTCPSend([]string{l.Addr().String()}, 0, 0)
		require.NoError(t, err)

		defer tcp.Close()

		err = tcp.Receive(m)
		require.NoError(t, err)

		wg.Wait()

		assert.Equal(t, m, recvMesg)
	})

	n.It("tries servers until it finds one that works", func() {
		l, err := net.Listen("tcp", ":0")
		require.NoError(t, err)

		defer l.Close()

		var recvMesg *cypress.Message

		var wg sync.WaitGroup

		wg.Add(1)
		go func() {
			defer wg.Done()

			c, err := l.Accept()
			require.NoError(t, err)

			recv, err := cypress.NewRecv(c)

			recvMesg, err = recv.Generate()
		}()

		m := cypress.Log()
		m.Add("hello", "world")

		addrs := []string{"127.0.0.1:45001", l.Addr().String(), "127.0.0.1:45001"}

		tcp, err := NewTCPSend(addrs, 0, 0)
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

			recv, err := cypress.NewRecv(c)
			require.NoError(t, err)

			<-latch

			_, err = recv.Generate()
			require.NoError(t, err)

			latch <- true
			<-latch
		}()

		m := cypress.Log()
		m.Add("hello", "world")

		tcp, err := NewTCPSend([]string{l.Addr().String()}, 0, 0)
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

		time.Sleep(100 * time.Millisecond)

		assert.Equal(t, 1, tcp.outstanding)
	})

	n.It("send nack'd messages once reconnected", func() {
		time.Sleep(1 * time.Second)

		l, err := net.Listen("tcp", ":0")
		require.NoError(t, err)

		defer l.Close()

		var recvMesg *cypress.Message
		var wg sync.WaitGroup

		latch := make(chan bool)

		wg.Add(1)
		go func() {
			defer wg.Done()

			c, err := l.Accept()
			require.NoError(t, err)

			defer c.Close()

			recv, err := cypress.NewRecv(c)
			require.NoError(t, err)

			<-latch

			_, err = recv.Generate()
			require.NoError(t, err)

			c.Close()

			c, err = l.Accept()
			require.NoError(t, err)

			defer c.Close()

			recv, err = cypress.NewRecv(c)
			require.NoError(t, err)

			recvMesg, err = recv.Generate()
			require.NoError(t, err)

			latch <- true
		}()

		m := cypress.Log()
		m.Add("hello", "world")

		tcp, err := NewTCPSend([]string{l.Addr().String()}, 0, 0)
		require.NoError(t, err)

		defer tcp.Close()

		err = tcp.Receive(m)
		require.NoError(t, err)

		m2 := cypress.Log()
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

	n.It("doesn't lose messages when the remote side is reconnected", func() {
		time.Sleep(1 * time.Second)

		l, err := net.Listen("tcp", ":0")
		require.NoError(t, err)

		addr := l.Addr().String()

		defer l.Close()

		var remote []*cypress.Message
		var wg sync.WaitGroup

		wg.Add(1)
		go func(l net.Listener) {
			defer wg.Done()

			c, err := l.Accept()
			require.NoError(t, err)

			recv, err := cypress.NewRecv(c)
			require.NoError(t, err)

			for i := 0; i < 50; i++ {
				m, err := recv.Generate()
				require.NoError(t, err)

				remote = append(remote, m)
			}

			time.Sleep(1 * time.Second)

			c.Close()

			c, err = l.Accept()
			require.NoError(t, err)

			defer c.Close()

			recv, err = cypress.NewRecv(c)
			require.NoError(t, err)

			for i := 0; i < 50; i++ {
				m, err := recv.Generate()
				require.NoError(t, err)

				remote = append(remote, m)
			}
		}(l)

		tcp, err := NewTCPSend([]string{addr}, 0, 0)
		require.NoError(t, err)

		defer tcp.Close()

		var sent []*cypress.Message

		for i := 0; i < 100; i++ {
			time.Sleep(1 * time.Millisecond)

			m := cypress.Log()
			m.Add("iter", i)

			sent = append(sent, m)

			err = tcp.Receive(m)
			require.NoError(t, err)
		}

		wg.Wait()

		for i := 0; i < 100; i++ {
			assert.NoError(t, sent[i].VerboseEqual(remote[i]))
		}
		for i := 0; i < 100; i++ {
			assert.NoError(t, sent[i].VerboseEqual(remote[i]))
		}
	})

	n.It("doesn't lose messages when seeing an error resending nack'd messages", func() {
		time.Sleep(1 * time.Second)

		l, err := net.Listen("tcp", ":0")
		require.NoError(t, err)

		addr := l.Addr().String()

		defer l.Close()

		var remote []*cypress.Message
		var wg sync.WaitGroup

		wg.Add(1)
		go func(l net.Listener) {
			defer wg.Done()

			c, err := l.Accept()
			require.NoError(t, err)

			recv, err := cypress.NewRecv(c)
			require.NoError(t, err)

			for i := 0; i < 50; i++ {
				m, err := recv.Generate()
				require.NoError(t, err)

				remote = append(remote, m)
			}

			time.Sleep(1 * time.Second)

			c.Close()

			c, err = l.Accept()
			require.NoError(t, err)

			recv, err = cypress.NewRecv(c)
			require.NoError(t, err)

			for i := 0; i < 10; i++ {
				m, err := recv.Generate()
				require.NoError(t, err)

				remote = append(remote, m)
			}

			c.Close()

			c, err = l.Accept()
			require.NoError(t, err)

			defer c.Close()

			recv, err = cypress.NewRecv(c)
			require.NoError(t, err)

			for i := 0; i < 40; i++ {
				m, err := recv.Generate()
				require.NoError(t, err)

				remote = append(remote, m)
			}
		}(l)

		tcp, err := NewTCPSend([]string{addr}, 0, 0)
		require.NoError(t, err)

		defer tcp.Close()

		var sent []*cypress.Message

		for i := 0; i < 100; i++ {
			time.Sleep(1 * time.Millisecond)

			m := cypress.Log()
			m.Add("iter", i)

			sent = append(sent, m)

			err = tcp.Receive(m)
			require.NoError(t, err)
		}

		wg.Wait()

		for i := 0; i < 100; i++ {
			assert.NoError(t, sent[i].VerboseEqual(remote[i]))
		}
	})
	n.Meow()
}

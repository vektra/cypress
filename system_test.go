package cypress

import (
	"io/ioutil"
	"net"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSendingLogsToSystem(t *testing.T) {
	dir, err := ioutil.TempDir("", "log")
	require.NoError(t, err)

	defer os.RemoveAll(dir)

	path := dir + "/" + "sock"

	l, err := net.Listen("unix", path)
	require.NoError(t, err)

	defer l.Close()

	buf := make([]byte, 10)

	m1 := Log()
	m1.Add("hello", "world")

	go func() {
		conn := ConnectTo(path)
		defer conn.Close()

		conn.Write(m1)
	}()

	c, err := l.Accept()
	require.NoError(t, err)

	n, err := c.Read(buf[:1])
	require.NoError(t, err)

	assert.Equal(t, n, 1)

	assert.Equal(t, "+", string(buf[:1]))

	m2 := &Message{}

	_, err = m2.ReadWire(c)
	require.NoError(t, err)

	assert.Equal(t, m1, m2)
}

func TestClientBuffersMessages(t *testing.T) {
	dir, err := ioutil.TempDir("", "log")
	require.NoError(t, err)

	defer os.RemoveAll(dir)

	path := dir + "/" + "sock"

	l, err := net.Listen("unix", path)
	require.NoError(t, err)

	buf := make([]byte, 10)

	m1 := Log()
	m1.Add("hello", "world")

	conn := ConnectTo(path)
	conn.Write(m1)

	c, err := l.Accept()
	require.NoError(t, err)

	_, err = c.Read(buf[:1])
	require.NoError(t, err)

	m2 := &Message{}

	_, err = m2.ReadWire(c)
	require.NoError(t, err)

	assert.Equal(t, m1, m2)

	c.Close()
	l.Close()

	m3 := Log()
	m3.Add("goodbye", "everyone")

	conn.Write(m3)

	time.Sleep(50 * time.Millisecond)

	l, err = net.Listen("unix", path)
	require.NoError(t, err)

	c, err = l.Accept()
	require.NoError(t, err)

	_, err = c.Read(buf[:1])
	require.NoError(t, err)

	m4 := &Message{}

	_, err = m4.ReadWire(c)
	require.NoError(t, err)

	assert.Equal(t, m3, m4)
}

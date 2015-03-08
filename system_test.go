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

	m1 := Log()
	m1.Add("hello", "world")

	go func() {
		conn := ConnectTo(path)
		defer conn.Close()

		conn.Write(m1)
	}()

	c, err := l.Accept()
	require.NoError(t, err)

	dec := NewDecoder(c)

	m2, err := dec.Decode()
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

	m1 := Log()
	m1.Add("hello", "world")

	conn := ConnectTo(path)
	conn.Write(m1)

	c, err := l.Accept()
	require.NoError(t, err)

	dec := NewDecoder(c)

	m2, err := dec.Decode()

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

	dec = NewDecoder(c)

	m4, err := dec.Decode()

	assert.Equal(t, m3, m4)
}

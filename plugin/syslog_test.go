package plugin

import (
	"bufio"
	"io/ioutil"
	"log/syslog"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vektra/cypress"
	"github.com/vektra/neko"
	"github.com/vektra/tai64n"
)

func TestSyslog(t *testing.T) {
	n := neko.Start(t)

	tmpdir, err := ioutil.TempDir("", "syslog")
	require.NoError(t, err)

	defer os.RemoveAll(tmpdir)

	n.It("parses a syslog message into a cypress Message", func() {
		line := "<14>2015-03-16T12:10:52-07:00 zero.local test[64480]: this is from golang tests\n"

		buf := bufio.NewReader(strings.NewReader(line))

		m, err := parseSyslog(buf)
		require.NoError(t, err)

		serv, ok := m.GetString("severity")
		require.True(t, ok)
		assert.Equal(t, "info", serv)

		fac, ok := m.GetString("facility")
		require.True(t, ok)

		assert.Equal(t, "user", fac)

		ts, err := time.Parse(time.RFC3339, "2015-03-16T12:10:52-07:00")
		require.NoError(t, err)

		assert.Equal(t, tai64n.FromTime(ts), m.GetTimestamp())

		host, ok := m.GetTag("host")
		require.True(t, ok)

		assert.Equal(t, "zero.local", host)

		tag, ok := m.GetString("tag")
		require.True(t, ok)

		assert.Equal(t, "test", tag)

		pid, ok := m.GetInt("pid")
		require.True(t, ok)

		assert.Equal(t, 64480, pid)

		msg, ok := m.GetString("message")
		require.True(t, ok)

		assert.Equal(t, "this is from golang tests", msg)
	})

	n.It("accepts data via a unix diagram socket and creates messages", func() {
		path := filepath.Join(tmpdir, "devlog")
		var buf cypress.BufferReceiver

		sl, err := NewSyslogDgram(path, &buf)
		require.NoError(t, err)

		sw, err := syslog.Dial("unixgram", path, syslog.LOG_INFO, "test")
		require.NoError(t, err)

		var wg sync.WaitGroup

		wg.Add(1)
		go func() {
			defer wg.Done()
			sl.Run()
		}()

		sw.Info("this is from golang tests")
		sw.Close()

		time.Sleep(100 * time.Millisecond)

		sl.Stop()

		wg.Wait()

		m := buf.Messages[0]

		sev, ok := m.GetString("severity")
		require.True(t, ok)

		assert.Equal(t, "info", sev)

		msg, ok := m.GetString("message")
		require.True(t, ok)

		assert.Equal(t, "this is from golang tests", msg)
	})

	n.It("accepts data via a tcp socket and creates messages", func() {
		var buf cypress.BufferReceiver

		tcp, err := net.Listen("tcp", ":0")
		require.NoError(t, err)

		defer tcp.Close()

		sl, err := NewSyslogFromListener(tcp, &buf)
		require.NoError(t, err)

		sw, err := syslog.Dial("tcp", tcp.Addr().String(), syslog.LOG_INFO, "test")
		require.NoError(t, err)

		var wg sync.WaitGroup

		wg.Add(1)
		go func() {
			defer wg.Done()
			sl.Run()
		}()

		sw.Info("this is from golang tests")
		sw.Close()

		time.Sleep(100 * time.Millisecond)

		sl.Stop()

		wg.Wait()

		m := buf.Messages[0]

		sev, ok := m.GetString("severity")
		require.True(t, ok)

		assert.Equal(t, "info", sev)

		msg, ok := m.GetString("message")
		require.True(t, ok)

		assert.Equal(t, "this is from golang tests", msg)
	})

	n.Meow()
}

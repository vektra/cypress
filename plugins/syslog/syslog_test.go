package syslog

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
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

		m, err := parseSyslog(buf, -1)
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

		assert.Equal(t, int64(64480), pid)

		msg, ok := m.GetString("message")
		require.True(t, ok)

		assert.Equal(t, "this is from golang tests", msg)
	})

	n.It("can parse a message when the timestamp is shorter than the format", func() {
		line := "<14>2015-03-16T12:10:52Z zero.local test[64480]: this is from golang tests\n"

		buf := bufio.NewReader(strings.NewReader(line))

		_, err := parseSyslog(buf, -1)
		require.NoError(t, err)
	})

	n.It("accepts data via a unix diagram socket and creates messages", func() {
		path := filepath.Join(tmpdir, "devlog")
		var buf cypress.BufferReceiver

		sl, err := NewSyslogDgram(path)
		require.NoError(t, err)

		sw, err := syslog.Dial("unixgram", path, syslog.LOG_INFO, "test")
		require.NoError(t, err)

		var wg sync.WaitGroup

		wg.Add(1)
		go func() {
			defer wg.Done()
			err := sl.Run(&buf)
			if err != nil {
				if !strings.Contains(err.Error(), "closed") {
					t.Fatalf("error running: %s", err)
				}
			}
		}()

		err = sw.Info("this is from golang tests")
		require.NoError(t, err)

		err = sw.Close()
		require.NoError(t, err)

		time.Sleep(1 * time.Second)

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

		sl, err := NewSyslogFromListener(tcp)
		require.NoError(t, err)

		sw, err := syslog.Dial("tcp", tcp.Addr().String(), syslog.LOG_INFO, "test")
		require.NoError(t, err)

		var wg sync.WaitGroup

		wg.Add(1)
		go func() {
			defer wg.Done()
			sl.Run(&buf)
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

	n.It("parses a RFC5424 syslog message into a cypress Message", func() {
		line := "<34>1 2003-10-11T22:14:15.003Z mymachine.example.com su - ID47 - 'su root' failed for lonvick on /dev/pts/8\n"

		buf := bufio.NewReader(strings.NewReader(line))

		m, err := parseSyslog(buf, -1)
		require.NoError(t, err)

		serv, ok := m.GetString("severity")
		require.True(t, ok)
		assert.Equal(t, "critical", serv)

		fac, ok := m.GetString("facility")
		require.True(t, ok)

		assert.Equal(t, "security", fac)

		ts, err := time.Parse(time.RFC3339, "2003-10-11T22:14:15.003Z")
		require.NoError(t, err)

		assert.Equal(t, tai64n.FromTime(ts), m.GetTimestamp())

		host, ok := m.GetTag("host")
		require.True(t, ok)

		assert.Equal(t, "mymachine.example.com", host)

		tag, ok := m.GetString("tag")
		require.True(t, ok)

		assert.Equal(t, "su", tag)

		msg, ok := m.GetString("message")
		require.True(t, ok)

		assert.Equal(t, "'su root' failed for lonvick on /dev/pts/8", msg)
	})

	n.It("parses a RFC5424 syslog message with a BOM marker", func() {
		line := "<34>1 2003-10-11T22:14:15.003Z mymachine.example.com su - ID47 - \xEF\xBB\xBF'su root' failed for lonvick on /dev/pts/8\n"

		buf := bufio.NewReader(strings.NewReader(line))

		m, err := parseSyslog(buf, -1)
		require.NoError(t, err)

		serv, ok := m.GetString("severity")
		require.True(t, ok)
		assert.Equal(t, "critical", serv)

		fac, ok := m.GetString("facility")
		require.True(t, ok)

		assert.Equal(t, "security", fac)

		ts, err := time.Parse(time.RFC3339, "2003-10-11T22:14:15.003Z")
		require.NoError(t, err)

		assert.Equal(t, tai64n.FromTime(ts), m.GetTimestamp())

		host, ok := m.GetTag("host")
		require.True(t, ok)

		assert.Equal(t, "mymachine.example.com", host)

		tag, ok := m.GetString("tag")
		require.True(t, ok)

		assert.Equal(t, "su", tag)

		msgid, ok := m.GetString("msgid")
		require.True(t, ok)

		assert.Equal(t, "ID47", msgid)

		msg, ok := m.GetString("message")
		require.True(t, ok)

		assert.Equal(t, "'su root' failed for lonvick on /dev/pts/8", msg)
	})

	n.It("parses a RFC5425 syslog message with structured data", func() {
		line := `<165>1 2003-10-11T22:14:15.003Z mymachine.example.com evntslog - ID47 [exampleSDID@32473 iut="3" eventSource="Application" eventID="1011"] ` +
			"\xEF\xBB\xBFAn application event log entry...\n"

		buf := bufio.NewReader(strings.NewReader(line))

		m, err := parseSyslog(buf, -1)
		require.NoError(t, err)

		serv, ok := m.GetString("severity")
		require.True(t, ok)
		assert.Equal(t, "notice", serv)

		fac, ok := m.GetString("facility")
		require.True(t, ok)

		assert.Equal(t, "local4", fac)

		ts, err := time.Parse(time.RFC3339, "2003-10-11T22:14:15.003Z")
		require.NoError(t, err)

		assert.Equal(t, tai64n.FromTime(ts), m.GetTimestamp())

		host, ok := m.GetTag("host")
		require.True(t, ok)

		assert.Equal(t, "mymachine.example.com", host)

		tag, ok := m.GetString("tag")
		require.True(t, ok)

		assert.Equal(t, "evntslog", tag)

		msgid, ok := m.GetString("msgid")
		require.True(t, ok)

		assert.Equal(t, "ID47", msgid)

		iut, ok := m.GetString("exampleSDID@32473.iut")
		require.True(t, ok)

		assert.Equal(t, "3", iut)

		src, ok := m.GetString("exampleSDID@32473.eventSource")
		require.True(t, ok)

		assert.Equal(t, "Application", src)

		eventId, ok := m.GetString("exampleSDID@32473.eventID")
		require.True(t, ok)

		assert.Equal(t, "1011", eventId)

		msg, ok := m.GetString("message")
		require.True(t, ok)

		assert.Equal(t, "An application event log entry...", msg)
	})

	n.It("can read octet-counted messages", func() {
		var buf bytes.Buffer

		emit := func(l string) {
			buf.WriteString(fmt.Sprintf("%d ", len(l)))
			buf.WriteString(l)
		}

		emit("<14>2015-03-16T12:10:52-07:00 zero.local test[64480]: this is from golang tests\n")
		emit("<14>2015-03-16T12:10:52-07:00 zero.local test[64481]: this is a second message\n")
		emit(`<165>1 2003-10-11T22:14:15.003Z mymachine.example.com evntslog - ID47 [exampleSDID@32473 iut="3" eventSource="Application" eventID="1011"] ` +
			"\xEF\xBB\xBFAn application event log entry...\n")

		var recv cypress.BufferReceiver

		s := &Syslog{OctetCounted: true}

		err := s.runConn(&buf, &recv)
		require.Equal(t, err, io.EOF)

		m1 := recv.Messages[0]
		m2 := recv.Messages[1]
		m3 := recv.Messages[2]

		pid1, ok := m1.GetInt("pid")
		require.True(t, ok)

		assert.Equal(t, int64(64480), pid1)

		msg1, ok := m1.GetString("message")
		require.True(t, ok)

		assert.Equal(t, "this is from golang tests", msg1)

		pid2, ok := m2.GetInt("pid")
		require.True(t, ok)

		assert.Equal(t, int64(64481), pid2)

		msg2, ok := m2.GetString("message")
		require.True(t, ok)

		assert.Equal(t, "this is a second message", msg2)

		msg3, ok := m3.GetString("message")
		require.True(t, ok)

		assert.Equal(t, "An application event log entry...", msg3)
	})

	n.Meow()
}

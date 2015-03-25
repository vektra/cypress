package syslog

import (
	"bufio"
	"bytes"
	"errors"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/vektra/cypress"
	"github.com/vektra/tai64n"
)

type Syslog struct {
	r cypress.Receiver

	c net.Conn
	l net.Listener
}

var facility = []string{
	"kernel",
	"user",
	"mail",
	"system",
	"security",
	"internal",
	"line printer",
	"network news",
	"UUCP",
	"clock",
	"security",
	"FTP",
	"NTP",
	"audit",
	"alert",
	"clock",
	"local0",
	"local1",
	"local2",
	"local3",
	"local4",
	"local5",
	"local6",
	"local7",
}

var severity = []string{
	"emergency",
	"alert",
	"critical",
	"error",
	"warning",
	"notice",
	"info",
	"debug",
}

func NewSyslogDgram(path string, r cypress.Receiver) (*Syslog, error) {
	unixAddr, err := net.ResolveUnixAddr("unixgram", path)
	if err != nil {
		return nil, err
	}

	c, err := net.ListenUnixgram("unixgram", unixAddr)
	if err != nil {
		return nil, err
	}

	return NewSyslogFromConn(c, r)
}

func NewSyslogFromListener(l net.Listener, r cypress.Receiver) (*Syslog, error) {
	return &Syslog{r: r, l: l}, nil
}

func NewSyslogFromConn(c net.Conn, r cypress.Receiver) (*Syslog, error) {
	return &Syslog{r: r, c: c}, nil
}

func (s *Syslog) runConn(c net.Conn) error {
	input := bufio.NewReader(c)

	for {
		m, err := parseSyslog(input)
		if err != nil {
			return err
		}

		err = s.r.Receive(m)
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *Syslog) Run() error {
	if s.c != nil {
		return s.runConn(s.c)
	}

	for {
		c, err := s.l.Accept()
		if err != nil {
			return err
		}

		go s.runConn(c)
	}
}

func (s *Syslog) Stop() error {
	if s.c != nil {
		return s.c.Close()
	}

	return s.l.Close()
}

var ErrInvalidFormat = errors.New("invalid format")

func parseSyslog(buf *bufio.Reader) (*cypress.Message, error) {
	c, err := buf.ReadByte()
	if err != nil {
		return nil, err
	}

	if c != '<' {
		return nil, ErrInvalidFormat
	}

	nxt, err := buf.ReadString('>')
	if err != nil {
		return nil, err
	}

	prio, err := strconv.Atoi(nxt[:len(nxt)-1])
	if err != nil {
		return nil, err
	}

	tsFmts := []string{
		"Jan 02 15:04:05",
		"Jan  2 15:04:05",
		time.RFC3339,
		time.Stamp,
	}

	var ts time.Time

	found := false
	for _, tsFmt := range tsFmts {
		tsFmtLen := len(tsFmt)

		sub, err := buf.Peek(tsFmtLen)
		if err != nil {
			return nil, err
		}

		if !strings.Contains(tsFmt, " ") {
			if idx := bytes.IndexByte(sub, ' '); idx != -1 {
				sub = sub[:idx]
			}
		}

		ts, err = time.Parse(tsFmt, string(sub))
		if err == nil {
			found = true
			buf.Read(sub)
			break
		}
	}

	if !found {
		return nil, ErrInvalidFormat
	}

	c, err = buf.ReadByte()
	if err != nil {
		return nil, err
	}

	if c != ' ' {
		return nil, ErrInvalidFormat
	}

	host, err := buf.ReadString(' ')
	if err != nil {
		return nil, err
	}

	host = host[:len(host)-1]

	tag, err := buf.ReadString(':')
	if err != nil {
		return nil, err
	}

	tag = tag[:len(tag)-1]

	var pid int

	if idx := strings.IndexByte(tag, '['); idx != -1 {
		if right := strings.IndexByte(tag, ']'); right != -1 {
			if p, err := strconv.Atoi(tag[idx+1 : right]); err == nil {
				pid = p
				tag = tag[:idx]
			}
		}
	}

	msg, err := buf.ReadString('\n')
	if err != nil {
		return nil, err
	}

	typ := uint32(cypress.LOG)

	m := &cypress.Message{
		Version:   cypress.DEFAULT_VERSION,
		Type:      &typ,
		Timestamp: tai64n.FromTime(ts),
	}

	m.AddTag("host", host)

	if prio > 8 {
		fac := prio / 8
		if fac > len(facility) {
			fac = 0
		}

		m.Add("facility", facility[fac])
		m.Add("severity", severity[prio%8])
	} else {
		m.Add("severity", severity[prio])
	}

	m.Add("tag", tag)
	if pid != 0 {
		m.Add("pid", pid)
	}
	m.Add("message", strings.TrimSpace(msg))

	return m, nil
}

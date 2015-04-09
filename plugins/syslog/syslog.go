package syslog

import (
	"bufio"
	"bytes"
	"errors"
	"io"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/vektra/cypress"
	"github.com/vektra/tai64n"
)

type Syslog struct {
	// Use RFC6587 encoded messages
	OctetCounted bool

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

func NewSyslogDgram(path string) (*Syslog, error) {
	unixAddr, err := net.ResolveUnixAddr("unixgram", path)
	if err != nil {
		return nil, err
	}

	c, err := net.ListenUnixgram("unixgram", unixAddr)
	if err != nil {
		return nil, err
	}

	return NewSyslogFromConn(c)
}

func NewSyslogTCP(addr string) (*Syslog, error) {
	l, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}

	s, err := NewSyslogFromListener(l)
	if err != nil {
		return nil, err
	}

	s.OctetCounted = true

	return s, err
}

func NewSyslogFromListener(l net.Listener) (*Syslog, error) {
	return &Syslog{l: l}, nil
}

func NewSyslogFromConn(c net.Conn) (*Syslog, error) {
	return &Syslog{c: c}, nil
}

func (s *Syslog) runConn(c io.Reader, r cypress.Receiver) error {
	input := bufio.NewReader(c)

	for {
		sz := -1

		if s.OctetCounted {
			szStr, err := input.ReadString(' ')
			if err != nil {
				return err
			}

			sz, err = strconv.Atoi(szStr[:len(szStr)-1])
			if err != nil {
				return err
			}
		}

		m, err := parseSyslog(input, sz)
		if err != nil {
			return err
		}

		err = r.Receive(m)
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *Syslog) Run(r cypress.Receiver) error {
	if s.c != nil {
		return s.runConn(s.c, r)
	}

	for {
		c, err := s.l.Accept()
		if err != nil {
			return err
		}

		go s.runConn(c, r)
	}
}

func (s *Syslog) Stop() error {
	if s.c != nil {
		return s.c.Close()
	}

	return s.l.Close()
}

var ErrInvalidFormat = errors.New("invalid format")

func parseSyslog(buf *bufio.Reader, total int) (*cypress.Message, error) {
	c, err := buf.ReadByte()
	if err != nil {
		return nil, err
	}

	if c != '<' {
		return nil, ErrInvalidFormat
	}

	total--

	nxt, err := buf.ReadString('>')
	if err != nil {
		return nil, err
	}

	total -= len(nxt)

	prio, err := strconv.Atoi(nxt[:len(nxt)-1])
	if err != nil {
		return nil, err
	}

	peek, err := buf.Peek(2)
	if err != nil {
		return nil, err
	}

	if peek[0] == '1' && peek[1] == ' ' {
		buf.ReadByte()
		buf.ReadByte()

		total -= 2
		return parse5425(prio, buf, total)
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
			n, _ := buf.Read(sub)

			total -= n
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

	total--

	host, err := buf.ReadString(' ')
	if err != nil {
		return nil, err
	}

	total -= len(host)

	host = host[:len(host)-1]

	tag, err := buf.ReadString(':')
	if err != nil {
		return nil, err
	}

	total -= len(tag)

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

	var msg string

	if total > 0 {
		msgBuf := make([]byte, total)
		_, err = buf.Read(msgBuf)
		if err != nil {
			return nil, err
		}

		msg = string(msgBuf)
	} else {
		msg, err = buf.ReadString('\n')
		if err != nil {
			return nil, err
		}
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

// <34>1 2003-10-11T22:14:15.003Z mymachine.example.com su - ID47 - 'su root' failed for lonvick on /dev/pts/8\n

func parse5425(prio int, buf *bufio.Reader, total int) (*cypress.Message, error) {
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
			n, _ := buf.Read(sub)
			total -= n
			break
		}
	}

	if !found {
		return nil, ErrInvalidFormat
	}

	c, err := buf.ReadByte()
	if err != nil {
		return nil, err
	}

	if c != ' ' {
		return nil, ErrInvalidFormat
	}

	total--

	host, err := buf.ReadString(' ')
	if err != nil {
		return nil, err
	}

	total -= len(host)

	host = host[:len(host)-1]

	tag, err := buf.ReadString(' ')
	if err != nil {
		return nil, err
	}

	total -= len(tag)

	tag = tag[:len(tag)-1]

	procid, err := buf.ReadString(' ')
	if err != nil {
		return nil, err
	}

	total -= len(procid)

	procid = procid[:len(procid)-1]

	msgid, err := buf.ReadString(' ')
	if err != nil {
		return nil, err
	}

	total -= len(msgid)

	msgid = msgid[:len(msgid)-1]

	typ := uint32(cypress.LOG)

	m := &cypress.Message{
		Version:   cypress.DEFAULT_VERSION,
		Type:      &typ,
		Timestamp: tai64n.FromTime(ts),
	}

	peek, err := buf.Peek(1)
	if err != nil {
		return nil, err
	}

	if peek[0] == '-' {
		buf.ReadByte()
		c, err := buf.ReadByte()
		if err != nil {
			return nil, err
		}

		if c != ' ' {
			return nil, ErrInvalidFormat
		}

		total -= 2
	} else {
		var valbuf bytes.Buffer

	outer:
		for {
			sdid, err := buf.ReadString(' ')
			if err != nil {
				return nil, err
			}

			total -= len(sdid)

			sdid = sdid[1 : len(sdid)-1]

			for {
				peek, err := buf.Peek(1)
				if err != nil {
					return nil, err
				}

				if peek[0] == ']' {
					buf.ReadByte()
					total--
					break outer
				}

				if peek[0] == ' ' {
					buf.ReadByte()
					total--
				}

				name, err := buf.ReadString('=')
				if err != nil {
					return nil, err
				}

				total -= len(name)

				name = name[:len(name)-1]

				q, err := buf.ReadByte()
				if err != nil {
					return nil, err
				}

				total--

				if q != '"' {
					return nil, ErrInvalidFormat
				}

				valbuf.Reset()

				for {
					c, err = buf.ReadByte()
					if err != nil {
						return nil, err
					}

					total--

					if c == '"' {
						break
					}

					if c == '\\' {
						c, err = buf.ReadByte()
						if err != nil {
							return nil, err
						}

						total--

						valbuf.WriteByte(c)
					} else {
						valbuf.WriteByte(c)
					}
				}

				val := valbuf.String()

				m.AddString(sdid+"."+name, val)
			}
		}
	}

	var msg string

	if total > 0 {
		msgBuf := make([]byte, total)

		_, err = buf.Read(msgBuf)
		if err != nil {
			return nil, err
		}

		msg = string(msgBuf)
	} else {
		msg, err = buf.ReadString('\n')
		if err != nil {
			return nil, err
		}
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
	m.Add("procid", procid)
	m.Add("msgid", msgid)

	msg = strings.TrimSpace(msg)

	if strings.HasPrefix(msg, "\xEF\xBB\xBF") {
		msg = msg[3:]
	}

	m.Add("message", msg)

	return m, nil

}

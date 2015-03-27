package syslog

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
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

	peek, err := buf.Peek(2)
	if err != nil {
		return nil, err
	}

	if peek[0] == '1' && peek[1] == ' ' {
		buf.ReadByte()
		buf.ReadByte()
		return parse5425(prio, buf)
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

// <34>1 2003-10-11T22:14:15.003Z mymachine.example.com su - ID47 - 'su root' failed for lonvick on /dev/pts/8\n

func parse5425(prio int, buf *bufio.Reader) (*cypress.Message, error) {
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

		fmt.Printf("sub: %s\n", string(sub))

		ts, err = time.Parse(tsFmt, string(sub))
		if err == nil {
			found = true
			buf.Read(sub)
			break
		}
	}

	fmt.Println("here 1")

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

	host, err := buf.ReadString(' ')
	if err != nil {
		return nil, err
	}

	host = host[:len(host)-1]

	tag, err := buf.ReadString(' ')
	if err != nil {
		return nil, err
	}

	tag = tag[:len(tag)-1]

	procid, err := buf.ReadString(' ')
	if err != nil {
		return nil, err
	}

	procid = procid[:len(procid)-1]

	msgid, err := buf.ReadString(' ')
	if err != nil {
		return nil, err
	}

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
	} else {
		var valbuf bytes.Buffer

	outer:
		for {
			fmt.Printf("TOP\n")
			sdid, err := buf.ReadString(' ')
			if err != nil {
				return nil, err
			}

			sdid = sdid[1 : len(sdid)-1]

			fmt.Printf("sdid: %s\n", sdid)

			for {
				peek, err := buf.Peek(1)
				if err != nil {
					return nil, err
				}

				if peek[0] == ']' {
					buf.ReadByte()
					break outer
				}

				if peek[0] == ' ' {
					buf.ReadByte()
				}

				name, err := buf.ReadString('=')
				if err != nil {
					return nil, err
				}

				name = name[:len(name)-1]

				fmt.Printf("  name: %s\n", name)

				q, err := buf.ReadByte()
				if err != nil {
					return nil, err
				}

				if q != '"' {
					return nil, ErrInvalidFormat
				}

				valbuf.Reset()

				for {
					c, err = buf.ReadByte()
					if err != nil {
						return nil, err
					}

					if c == '"' {
						break
					}

					if c == '\\' {
						c, err = buf.ReadByte()
						if err != nil {
							return nil, err
						}

						valbuf.WriteByte(c)
					} else {
						valbuf.WriteByte(c)
					}
				}

				val := valbuf.String()

				fmt.Printf("  val: %s\n", val)

				m.AddString(sdid+"."+name, val)
			}
		}
	}

	msg, err := buf.ReadString('\n')
	if err != nil {
		return nil, err
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

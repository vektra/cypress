package syslog

import (
	"fmt"
	"net"

	"github.com/vektra/cypress"
)

type Plugin struct {
	Dgram string
	TCP   string
	UDP   string

	OctetCounted bool
}

func (s *Plugin) Generator() (cypress.Generator, error) {
	var cnt int

	if s.Dgram != "" {
		cnt++
	}

	if s.TCP != "" {
		cnt++
	}

	if s.UDP != "" {
		cnt++
	}

	var (
		conn *Syslog
		err  error
	)

	switch {
	case cnt == 0:
		return nil, fmt.Errorf("specify a method for receiving syslog message")
	case cnt > 1:
		return nil, fmt.Errorf("specify only one method")
	case s.Dgram != "":
		conn, err = NewSyslogDgram(s.Dgram)
		if err != nil {
			return nil, err
		}

	case s.TCP != "":
		l, err := net.Listen("tcp", s.TCP)
		if err != nil {
			return nil, err
		}

		conn, err = NewSyslogFromListener(l)
		if err != nil {
			return nil, err
		}

		conn.OctetCounted = s.OctetCounted
	case s.UDP != "":
		addr, err := net.ResolveUDPAddr("udp", s.UDP)
		if err != nil {
			return nil, err
		}

		c, err := net.ListenUDP("udp", addr)
		if err != nil {
			return nil, err
		}

		conn, err = NewSyslogFromConn(c)
		if err != nil {
			return nil, err
		}
	}

	c := make(cypress.Channel, 1)

	go conn.Run(c)

	return c, nil
}

func init() {
	cypress.AddPlugin("syslog", func() cypress.Plugin { return &Plugin{} })
}

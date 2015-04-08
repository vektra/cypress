package syslog

import (
	"fmt"
	"net"
	"os"

	"github.com/vektra/cypress"
	"github.com/vektra/cypress/cli/commands"
)

type CLI struct {
	Dgram string `long:"dgram" description:"Listen for unix diagrams at a path"`
	TCP   string `short:"t" long:"tcp" description:"Listen on a TCP port"`
	UDP   string `short:"u" long:"udp" description:"Listen on a UDP port"`

	OctetCounted bool `short:"c" long:"octet-counted" default:"true" description:"For TCP, use RFC6587 encoded messages"`
}

func (s *CLI) Execute(args []string) error {
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

	r := cypress.NewStreamEncoder(os.Stdout)
	err := r.Init(cypress.SNAPPY)
	if err != nil {
		return err
	}

	var conn *Syslog

	switch {
	case cnt == 0:
		return fmt.Errorf("specify a method for receiving syslog message")
	case cnt > 1:
		return fmt.Errorf("specify only one method")
	case s.Dgram != "":
		conn, err = NewSyslogDgram(s.Dgram, r)
	case s.TCP != "":
		l, err := net.Listen("tcp", s.TCP)
		if err != nil {
			return err
		}

		conn, err = NewSyslogFromListener(l, r)

		conn.OctetCounted = s.OctetCounted
	case s.UDP != "":
		addr, err := net.ResolveUDPAddr("udp", s.UDP)
		if err != nil {
			return err
		}

		c, err := net.ListenUDP("udp", addr)
		if err != nil {
			return err
		}

		conn, err = NewSyslogFromConn(c, r)
	}

	return conn.Run()
}

func init() {
	commands.Add("syslog", "Receive syslog messages", "", &CLI{})
}

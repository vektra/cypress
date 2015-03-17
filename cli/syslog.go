package cli

import (
	"fmt"
	"net"
	"os"

	"github.com/vektra/cypress"
	"github.com/vektra/cypress/plugin"
)

type Syslog struct {
	Dgram string `long:"dgram" description:"Listen for unix diagrams at a path"`
	TCP   string `short:"t" long:"tcp" description:"Listen on a TCP port"`
	UDP   string `short:"u" long:"udp" description:"Listen on a UDP port"`
}

func (s *Syslog) Execute(args []string) error {
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

	var syslog *plugin.Syslog

	switch {
	case cnt == 0:
		return fmt.Errorf("specify a method for receiving syslog message")
	case cnt > 1:
		return fmt.Errorf("specify only one method")
	case s.Dgram != "":
		syslog, err = plugin.NewSyslogDgram(s.Dgram, r)
	case s.TCP != "":
		l, err := net.Listen("tcp", s.TCP)
		if err != nil {
			return err
		}

		syslog, err = plugin.NewSyslogFromListener(l, r)
	case s.UDP != "":
		addr, err := net.ResolveUDPAddr("udp", s.UDP)
		if err != nil {
			return err
		}

		c, err := net.ListenUDP("udp", addr)
		if err != nil {
			return err
		}

		syslog, err = plugin.NewSyslogFromConn(c, r)
	}

	return syslog.Run()
}

func init() {
	addCommand("syslog", "Receive syslog messages", "", &Syslog{})
}

package tcplog

import (
	"crypto/rand"
	"fmt"
	"net"
	"testing"

	"github.com/vektra/cypress"
)

func randString(n int) string {
	const alphanum = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz "
	var bytes = make([]byte, n)
	rand.Read(bytes)
	for i, b := range bytes {
		bytes[i] = alphanum[b%byte(len(alphanum))]
	}
	return string(bytes)
}

func NewMessage(t *testing.T) *cypress.Message {
	message := cypress.Log()
	err := message.Add("message", randString(50))
	err = message.AddString("string_key", "I'm a string!")
	err = message.AddInt("int_key", 12)
	err = message.AddBytes("bytes_key", []byte("I'm bytes!"))
	err = message.AddInterval("interval_key", 2, 1)
	if err != nil {
		t.Errorf("Error adding message: %s", err)
	}
	return message
}

type TcpServer struct {
	Port     int
	Address  chan string
	Messages chan []byte
}

func NewTcpServer() *TcpServer {
	return &TcpServer{
		Address:  make(chan string, 1),
		Messages: make(chan []byte, 1),
	}
}

func (s *TcpServer) Run(host string) {
	var (
		ln  net.Listener
		err error
	)

	ln, err = net.Listen("tcp", "")

	if err != nil {
		fmt.Println(err)
		return
	}

	s.Address <- fmt.Sprintf("%s:%d", host, ln.Addr().(*net.TCPAddr).Port)

	conn, err := ln.Accept()

	if err != nil {
		fmt.Println(err)
		return
	}

	defer conn.Close()

	s.handleConnection(conn)
}

func (s *TcpServer) handleConnection(conn net.Conn) {
	buf := make([]byte, 1024)

	reqLen, err := conn.Read(buf)

	if err != nil {
		fmt.Println(err)
		return
	}

	s.Messages <- buf[0:reqLen]
}

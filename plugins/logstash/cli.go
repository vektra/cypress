package logstash

import (
	"fmt"
	"os"
	"sync"

	"github.com/vektra/cypress"
	"github.com/vektra/cypress/cli/commands"
)

type Send struct {
	Host string `short:"H" long:"host" description:"Logstash host <host>:<port>"`
	Port string `short:"P" long:"port" description:"Logstash port <host>:<port>"`
	Ssl  bool   `short:"S" long:"tls" default:"false" description:"Use TLS"`
}

func (p *Send) Execute(args []string) error {
	address := fmt.Sprintf("%s:%s", p.Host, p.Port)

	logstash := NewLogger(address, p.Ssl)

	dec, err := cypress.NewStreamDecoder(os.Stdin)
	if err != nil {
		return err
	}

	var wg sync.WaitGroup

	defer wg.Wait()

	wg.Add(1)

	go func() {
		defer wg.Done()

		logstash.Run()
	}()

	defer logstash.Close()

	return cypress.Glue(dec, logstash)
}

func init() {
	short := "Send messages to Logstash"
	long := "Given a stream on stdin, the logstash command will read those messages in and send them to Logstash via TCP."

	commands.Add("logstash", short, long, &Send{})
}

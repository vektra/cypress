package cli

import (
	"os"
	"sync"

	"github.com/vektra/cypress"
)

type Recv struct {
	Listen string `short:"l" long:"listen" description:"host:port to listen on"`

	lock sync.Mutex

	out cypress.Receiver
}

func (r *Recv) Execute(args []string) error {
	tcp, err := cypress.NewTCPRecv(r.Listen, r)
	if err != nil {
		return err
	}

	r.out = cypress.NewSerialReceiver(cypress.NewStreamEncoder(os.Stdout))

	return tcp.ListenAndAccept()
}

func (r *Recv) HandleGenerator(gen cypress.Generator) {
	cypress.Glue(gen, r.out)
}

func init() {
	addCommand("recv", "accept streams over the network", "", &Recv{})
}

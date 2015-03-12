package plugin

import (
	"github.com/vektra/cypress"
	"github.com/vektra/cypress/statsd"
)

type Statsd struct {
	out cypress.Receiver

	Server *statsd.Server
}

func NewStatsdEndpoint(out cypress.Receiver, address string) (*Statsd, error) {
	s := &Statsd{out: out}

	serv := &statsd.Server{
		Addr:    address,
		Handler: statsd.HandlerFunc(s.handleMetric),
	}

	err := serv.Listen()
	if err != nil {
		return nil, err
	}

	s.Server = serv

	return s, nil
}

func (s *Statsd) handleMetric(sm *statsd.Metric) {
	m := cypress.Metric()
	m.AddString("name", sm.Bucket)
	m.AddString("type", sm.Type.String())
	m.AddFloat("value", sm.Value)

	s.out.Receive(m)
}

func (s *Statsd) Run() error {
	return s.Server.ListenAndReceive()
}

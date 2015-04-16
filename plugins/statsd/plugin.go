package statsd

import "github.com/vektra/cypress"

type Plugin struct {
	Listen string
}

func (p *Plugin) Generator() (cypress.Generator, error) {
	c := make(cypress.Channel, 1)

	ep, err := NewStatsdEndpoint(c, p.Listen)
	if err != nil {
		return nil, err
	}

	go ep.Run()

	return c, nil
}

func init() {
	cypress.AddPlugin("statsd", func() cypress.Plugin { return &Plugin{} })
}

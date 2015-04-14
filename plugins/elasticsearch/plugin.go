package elasticsearch

import "github.com/vektra/cypress"

type Plugin struct {
	Host     string
	Index    string
	Prefix   string
	Logstash bool
}

func (p *Plugin) Receiver() (cypress.Receiver, error) {
	s := &Store{
		Host:     p.Host,
		Index:    p.Index,
		Prefix:   p.Prefix,
		Logstash: p.Logstash,
	}

	err := s.Init()
	if err != nil {
		return nil, err
	}

	return s, nil
}

func init() {
	cypress.AddPlugin("elasticsearch", func() cypress.Plugin { return &Plugin{} })
}

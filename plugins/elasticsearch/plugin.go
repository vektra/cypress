package elasticsearch

import "github.com/vektra/cypress"

type Plugin struct {
	Host     string `description:"host:port of elasticsearch node"`
	Index    string `description:"fixed index to use if set"`
	Prefix   string `description:"index prefix to use with dated indices (cypress)"`
	Logstash bool   `description:"use logstash format indices"`
}

func (p *Plugin) Description() string {
	return `Write messages to Elasticsearch. By default, indices are generated per day with the prefix of 'cypress'`
}

func (p *Plugin) Receiver() (cypress.Receiver, error) {
	s := &Send{
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

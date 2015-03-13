package commands

import "github.com/vektra/cypress/plugin"

type Metrics struct {
	httpAddr string
	statsd   *plugin.Statsd
	sink     *plugin.MetricSink
}

func NewMetrics(statsd, http string) (*Metrics, error) {
	ms := plugin.NewMetricSink()

	se, err := plugin.NewStatsdEndpoint(ms, statsd)
	if err != nil {
		return nil, err
	}

	cmd := &Metrics{
		httpAddr: http,
		sink:     ms,
		statsd:   se,
	}

	return cmd, nil
}

func (m *Metrics) Run() error {
	go m.sink.RunHTTP(m.httpAddr)
	return m.statsd.Run()
}

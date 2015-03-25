package commands

import (
	"github.com/vektra/cypress/plugins/metrics"
	"github.com/vektra/cypress/plugins/statsd"
)

type Metrics struct {
	httpAddr string
	statsd   *statsd.Statsd
	sink     *metrics.MetricSink
}

func NewMetrics(addr, http string) (*Metrics, error) {
	ms := metrics.NewMetricSink()

	se, err := statsd.NewStatsdEndpoint(ms, addr)
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

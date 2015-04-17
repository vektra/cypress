package metrics

import (
	"io/ioutil"
	"log"

	"github.com/naoina/toml"
	"github.com/vektra/cypress"
)

type Plugin struct {
	Listen string `description:"host:port to run an internal HTTP server on"`
	Config string `description:"path to the metrics config file"`
}

func (p *Plugin) Description() string {
	return `Metrics aggregator. Provides HTTP to query aggregation.`
}

func (p *Plugin) Receiver() (cypress.Receiver, error) {
	var mc MetricsConfig

	mc.Influx = DefaultInfluxConfig()

	mc.HTTP = p.Listen

	if p.Config != "" {
		data, err := ioutil.ReadFile(p.Config)
		if err != nil {
			return nil, err
		}

		err = toml.Unmarshal(data, &mc)
		if err != nil {
			return nil, err
		}
	}

	sink := NewMetricSink()

	if p.Listen != "" {
		log.Printf("Started HTTP server at %s", p.Listen)
		go sink.RunHTTP(p.Listen)
	}

	if mc.Influx.URL != "" {
		err := sink.EnableInflux(mc.Influx)
		if err != nil {
			return nil, err
		}
	}

	return sink, nil
}

func init() {
	cypress.AddPlugin("metrics", func() cypress.Plugin { return &Plugin{} })
}

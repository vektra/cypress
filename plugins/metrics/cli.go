package metrics

import (
	"io/ioutil"
	"log"
	"os"

	"github.com/naoina/toml"
	"github.com/vektra/cypress"
	"github.com/vektra/cypress/cli/commands"
)

type MetricsConfig struct {
	HTTP   string
	Influx *InfluxConfig
}

type Metrics struct {
	HTTP   string `short:"l" long:"listen" description:"Port to run metrics HTTP service on"`
	Config string `short:"c" long:"config" description:"Configuration file of metrics"`
}

func (m *Metrics) Execute(args []string) error {
	var mc MetricsConfig

	mc.Influx = DefaultInfluxConfig()

	mc.HTTP = m.HTTP

	if m.Config != "" {
		data, err := ioutil.ReadFile(m.Config)
		if err != nil {
			return err
		}

		err = toml.Unmarshal(data, &mc)
		if err != nil {
			return err
		}
	}

	sink := NewMetricSink()

	dec, err := cypress.NewStreamDecoder(os.Stdin)
	if err != nil {
		return err
	}

	if m.HTTP != "" {
		log.Printf("Started HTTP server at %s", m.HTTP)
		go sink.RunHTTP(m.HTTP)
	}

	if mc.Influx.URL != "" {
		err := sink.EnableInflux(mc.Influx)
		if err != nil {
			return err
		}

		log.Printf("Enabled InfluxDB exporter to %s", mc.Influx.URL)

		commands.OnShutdown(func() {
			log.Printf("Flushing data to InfluxDB...")
			sink.FlushInflux(mc.Influx)
		})
	}

	return cypress.Glue(dec, sink)
}

type Plugin struct {
	Listen string
	Config string
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
	commands.Add("metrics", "collect metrics", "", &Metrics{})
	cypress.AddPlugin("metrics", func() cypress.Plugin { return &Plugin{} })
}

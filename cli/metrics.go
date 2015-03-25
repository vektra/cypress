package cli

import (
	"io/ioutil"
	"log"
	"os"

	"github.com/naoina/toml"
	"github.com/vektra/cypress"
	"github.com/vektra/cypress/plugins/metrics"
)

type MetricsConfig struct {
	HTTP   string
	Influx *metrics.InfluxConfig
}

type Metrics struct {
	HTTP   string `short:"H" long:"http" description:"Port to run metrics HTTP service on"`
	Config string `short:"c" long:"config" description:"Configuration file of metrics"`
}

func (m *Metrics) Execute(args []string) error {
	var mc MetricsConfig

	mc.Influx = metrics.DefaultInfluxConfig()

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

	metrics := metrics.NewMetricSink()

	dec, err := cypress.NewStreamDecoder(os.Stdin)
	if err != nil {
		return err
	}

	if m.HTTP != "" {
		log.Printf("Started HTTP server at %s", m.HTTP)
		go metrics.RunHTTP(m.HTTP)
	}

	if mc.Influx.URL != "" {
		err := metrics.EnableInflux(mc.Influx)
		if err != nil {
			return err
		}

		log.Printf("Enabled InfluxDB exporter to %s", mc.Influx.URL)

		Lifecycle.OnShutdown(func() {
			log.Printf("Flushing data to InfluxDB...")
			metrics.FlushInflux(mc.Influx)
		})
	}

	return cypress.Glue(dec, metrics)
}

func init() {
	addCommand("metrics", "collect metrics", "", &Metrics{})
}

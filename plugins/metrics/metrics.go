package metrics

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/rcrowley/go-metrics"
	"github.com/vektra/cypress"
	"github.com/vektra/cypress/influxdb"
)

type MetricSink struct {
	Registry metrics.Registry
}

type InfluxConfig struct {
	Flush     cypress.Duration
	URL       string
	Username  string
	Password  string
	Database  string
	UserAgent string
}

const (
	DefaultFlushDuration  = "10s"
	DefaultUserAgent      = "cypress/1"
	DefaultInfluxUsername = "cypress"
)

func DefaultInfluxConfig() *InfluxConfig {
	dur, err := time.ParseDuration(DefaultFlushDuration)
	if err != nil {
		panic(err)
	}
	return &InfluxConfig{
		Flush:     cypress.Duration{dur},
		UserAgent: DefaultUserAgent,
		Username:  DefaultInfluxUsername,
	}
}

func NewMetricSink() *MetricSink {
	return &MetricSink{Registry: metrics.NewRegistry()}
}

func (ms *MetricSink) RunHTTP(addr string) error {
	serv := http.Server{
		Addr:    addr,
		Handler: http.HandlerFunc(ms.outputMetrics),
	}

	return serv.ListenAndServe()
}

func (cfg *InfluxConfig) Export() *influxdb.Config {
	u, err := url.Parse(fmt.Sprintf("%s/db/%s/series?u=%s&p=%s",
		cfg.URL, cfg.Database, cfg.Username, cfg.Password))

	if err != nil {
		panic(err)
	}

	return &influxdb.Config{
		URL:       *u,
		Username:  cfg.Username,
		Password:  cfg.Password,
		UserAgent: cfg.UserAgent,
		Database:  cfg.Database,
	}
}

func (ms *MetricSink) EnableInflux(cfg *InfluxConfig) error {
	xcfg := cfg.Export()

	go influxdb.Influxdb(ms.Registry, cfg.Flush.Duration, xcfg)

	return nil
}

func (ms *MetricSink) FlushInflux(cfg *InfluxConfig) error {
	xcfg := cfg.Export()

	influxdb.Influxdb(ms.Registry, 0, xcfg)

	return nil
}

func (ms *MetricSink) outputMetrics(res http.ResponseWriter, req *http.Request) {
	json.NewEncoder(res).Encode(ms.Registry)
}

var ErrInvalidMetric = errors.New("invalid metric")

func (ms *MetricSink) Receive(m *cypress.Message) error {
	if m.GetType() != cypress.METRIC {
		return nil
	}

	typ, ok := m.GetString("type")
	if !ok {
		return ErrInvalidMetric
	}

	name, ok := m.GetString("name")
	if !ok {
		return ErrInvalidMetric
	}

	value, ok := m.Get("value")
	if !ok {
		return ErrInvalidMetric
	}

	switch typ {
	case "counter":
		var ival int64

		switch sv := value.(type) {
		case int64:
			ival = sv
		case float64:
			ival = int64(sv)
		default:
			return ErrInvalidMetric
		}

		metrics.GetOrRegisterCounter(name, ms.Registry).Inc(ival)
	case "gauge":
		var fval float64

		switch sv := value.(type) {
		case int64:
			fval = float64(sv)
		case float64:
			fval = sv
		default:
			return ErrInvalidMetric
		}

		metrics.GetOrRegisterGaugeFloat64(name, ms.Registry).Update(fval)
	case "timer":
		interval, ok := m.GetInterval("value")
		if !ok {
			return ErrInvalidMetric
		}

		metrics.GetOrRegisterTimer(name, ms.Registry).Update(interval.Duration())
	default:
		return ErrInvalidMetric
	}

	return nil
}

func (ms *MetricSink) Close() error {
	return nil
}

package plugin

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/rcrowley/go-metrics"
	"github.com/vektra/cypress"
)

type MetricSink struct {
	Registry metrics.Registry
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

func (ms *MetricSink) outputMetrics(res http.ResponseWriter, req *http.Request) {
	json.NewEncoder(res).Encode(ms.Registry)
}

var ErrInvalidMetric = errors.New("invalid metric")

func (ms *MetricSink) Receive(m *cypress.Message) error {
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

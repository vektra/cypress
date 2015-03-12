package statsd

import "fmt"

// Lovingly imported from https://github.com/kisielk/gostatsd

// MetricType is an enumeration of all the possible types of Metric
type MetricType int

const (
	COUNTER MetricType = iota
	TIMER
	GAUGE
	GAUGE_DELTA
	SET
)

func (m MetricType) String() string {
	switch m {
	case GAUGE:
		return "gauge"
	case GAUGE_DELTA:
		return "gauge_delta"
	case TIMER:
		return "timer"
	case COUNTER:
		return "counter"
	default:
		return "unknown"
	}
}

// Metric represents a single data collected datapoint
type Metric struct {
	Type   MetricType // The type of metric
	Bucket string     // The name of the bucket where the metric belongs
	Value  float64    // The numeric value of the metric
}

func (m Metric) String() string {
	return fmt.Sprintf("{%s, %s, %f}", m.Type, m.Bucket, m.Value)
}

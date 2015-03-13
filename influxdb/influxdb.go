package influxdb

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"time"

	influxClient "github.com/influxdb/influxdb/client"
	"github.com/rcrowley/go-metrics"
)

type Config struct {
	URL       url.URL
	Database  string
	Username  string
	Password  string
	UserAgent string
}

func Influxdb09(r metrics.Registry, d time.Duration, config *Config) {
	client, err := influxClient.NewClient(influxClient.Config{
		URL:       config.URL,
		Username:  config.Username,
		Password:  config.Password,
		UserAgent: config.UserAgent,
	})

	if err != nil {
		log.Println(err)
		return
	}

	fmt.Printf("flushing to influx every %s\n", d)

	for _ = range time.Tick(d) {
		fmt.Printf("sending to influx...\n")
		if err := send09(r, config.Database, client); err != nil {
			log.Println(err)
		}
	}
}

func send09(r metrics.Registry, db string, client *influxClient.Client) error {
	bp := influxClient.BatchPoints{
		Database: db,
	}

	r.Each(func(name string, i interface{}) {
		now := getCurrentTime()
		switch metric := i.(type) {
		case metrics.Counter:
			bp.Points = append(bp.Points, influxClient.Point{
				Name: fmt.Sprintf("%s.count", name),
				Fields: map[string]interface{}{
					"time":  now,
					"count": metric.Count(),
				},
			})
		case metrics.Gauge:
			bp.Points = append(bp.Points, influxClient.Point{
				Name: fmt.Sprintf("%s.value", name),
				Fields: map[string]interface{}{
					"time":  now,
					"value": metric.Value(),
				},
			})
		case metrics.GaugeFloat64:
			bp.Points = append(bp.Points, influxClient.Point{
				Name: fmt.Sprintf("%s.value", name),
				Fields: map[string]interface{}{
					"time":  now,
					"value": metric.Value(),
				},
			})
		case metrics.Histogram:
			h := metric.Snapshot()
			ps := h.Percentiles([]float64{0.5, 0.75, 0.95, 0.99, 0.999})

			bp.Points = append(bp.Points, influxClient.Point{
				Name: fmt.Sprintf("%s.histogram", name),
				Fields: map[string]interface{}{
					"time":           now,
					"count":          h.Count(),
					"min":            h.Min(),
					"max":            h.Max(),
					"mean":           h.Mean(),
					"std-dev":        h.StdDev(),
					"50-percentile":  ps[0],
					"75-percentile":  ps[1],
					"95-percentile":  ps[2],
					"99-percentile":  ps[3],
					"999-percentile": ps[4],
				},
			})
		case metrics.Meter:
			m := metric.Snapshot()
			bp.Points = append(bp.Points, influxClient.Point{
				Name: fmt.Sprintf("%s.meter", name),
				Fields: map[string]interface{}{
					"count":          m.Count(),
					"one-minute":     m.Rate1(),
					"five-minute":    m.Rate5(),
					"fifteen-minute": m.Rate15(),
					"mean":           m.RateMean(),
				},
			})
		case metrics.Timer:
			h := metric.Snapshot()
			ps := h.Percentiles([]float64{0.5, 0.75, 0.95, 0.99, 0.999})
			bp.Points = append(bp.Points, influxClient.Point{
				Name: fmt.Sprintf("%s.timer", name),
				Fields: map[string]interface{}{
					"count":          h.Count(),
					"min":            h.Min(),
					"max":            h.Max(),
					"mean":           h.Mean(),
					"std-dev":        h.StdDev(),
					"50-percentile":  ps[0],
					"75-percentile":  ps[1],
					"95-percentile":  ps[2],
					"99-percentile":  ps[3],
					"999-percentile": ps[4],
					"one-minute":     h.Rate1(),
					"five-minute":    h.Rate5(),
					"fifteen-minute": h.Rate15(),
					"mean-rate":      h.RateMean(),
				},
			})
		}
	})

	_, err := client.Write(bp)
	if err != nil {
		log.Println(err)
	}

	return nil
}

func getCurrentTime() int64 {
	return time.Now().UnixNano() / 1000000
}

func Influxdb(r metrics.Registry, d time.Duration, config *Config) {
	if d == 0 {
		if err := send08(r, config); err != nil {
			log.Println(err)
		}

		return
	}

	for _ = range time.Tick(d) {
		if err := send08(r, config); err != nil {
			log.Println(err)
		}
	}
}

type Series struct {
	Name    string   `json:"name"`
	Columns []string `json:"columns"`
	Points  [][]interface{}
}

func send08(r metrics.Registry, config *Config) error {
	series := []*Series{}

	r.Each(func(name string, i interface{}) {
		now := getCurrentTime()
		switch metric := i.(type) {
		case metrics.Counter:
			series = append(series, &Series{
				Name:    fmt.Sprintf("%s.count", name),
				Columns: []string{"time", "count"},
				Points: [][]interface{}{
					{now, metric.Count()},
				},
			})
		case metrics.Gauge:
			series = append(series, &Series{
				Name:    fmt.Sprintf("%s.value", name),
				Columns: []string{"time", "value"},
				Points: [][]interface{}{
					{now, metric.Value()},
				},
			})
		case metrics.GaugeFloat64:
			series = append(series, &Series{
				Name:    fmt.Sprintf("%s.value", name),
				Columns: []string{"time", "value"},
				Points: [][]interface{}{
					{now, metric.Value()},
				},
			})
		case metrics.Histogram:
			h := metric.Snapshot()
			ps := h.Percentiles([]float64{0.5, 0.75, 0.95, 0.99, 0.999})
			series = append(series, &Series{
				Name: fmt.Sprintf("%s.histogram", name),
				Columns: []string{"time", "count", "min", "max", "mean", "std-dev",
					"50-percentile", "75-percentile", "95-percentile",
					"99-percentile", "999-percentile"},
				Points: [][]interface{}{
					{now, h.Count(), h.Min(), h.Max(), h.Mean(), h.StdDev(),
						ps[0], ps[1], ps[2], ps[3], ps[4]},
				},
			})
		case metrics.Meter:
			m := metric.Snapshot()
			series = append(series, &Series{
				Name: fmt.Sprintf("%s.meter", name),
				Columns: []string{"count", "one-minute",
					"five-minute", "fifteen-minute", "mean"},
				Points: [][]interface{}{
					{m.Count(), m.Rate1(), m.Rate5(), m.Rate15(), m.RateMean()},
				},
			})
		case metrics.Timer:
			h := metric.Snapshot()
			ps := h.Percentiles([]float64{0.5, 0.75, 0.95, 0.99, 0.999})
			series = append(series, &Series{
				Name: fmt.Sprintf("%s.timer", name),
				Columns: []string{"count", "min", "max", "mean", "std-dev",
					"50-percentile", "75-percentile", "95-percentile",
					"99-percentile", "999-percentile", "one-minute", "five-minute", "fifteen-minute", "mean-rate"},
				Points: [][]interface{}{
					{h.Count(), h.Min(), h.Max(), h.Mean(), h.StdDev(),
						ps[0], ps[1], ps[2], ps[3], ps[4],
						h.Rate1(), h.Rate5(), h.Rate15(), h.RateMean()},
				},
			})
		}
	})

	var buf bytes.Buffer

	err := json.NewEncoder(&buf).Encode(series)
	if err != nil {
		log.Println(err)
		return nil
	}

	req, err := http.NewRequest("POST", config.URL.String(), &buf)
	if err != nil {
		log.Println(err)
		return nil
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Println(err)
		return nil
	}

	resp.Body.Close()

	if resp.StatusCode != 200 {
		log.Printf("Response indicated error: %d\n", resp.StatusCode)
	}

	return nil
}

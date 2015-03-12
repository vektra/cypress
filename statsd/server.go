package statsd

// Lovingly inspired by https://github.com/kisielk/gostatsd

import (
	"bytes"
	"errors"
	"fmt"
	"log"
	"net"
	"strconv"
	"strings"
)

// DefaultMetricsAddr is the default address on which a MetricReceiver will listen
const DefaultMetricsAddr = ":8125"

// Objects implementing the Handler interface can be used to handle metrics for
// a Server
type Handler interface {
	HandleMetric(m *Metric)
}

// The HandlerFunc type is an adapter to allow the use of ordinary functions as
// metric handlers
type HandlerFunc func(*Metric)

// HandleMetric calls f(m)
func (f HandlerFunc) HandleMetric(m *Metric) {
	f(m)
}

// Server receives data on its listening port and converts lines in to Metrics.
// For each Metric it calls r.Handler.HandleMetric()
type Server struct {
	Addr    string  // UDP address on which to listen for metrics
	Handler Handler // handler to invoke

	c net.PacketConn
}

func (r *Server) Listen() error {
	if r.c != nil {
		return nil
	}

	addr := r.Addr
	if addr == "" {
		addr = DefaultMetricsAddr
	}

	c, err := net.ListenPacket("udp", addr)
	if err != nil {
		return err
	}

	if r.Addr == ":0" {
		r.Addr = c.LocalAddr().String()
	}

	r.c = c

	return nil
}

// ListenAndReceive listens on the UDP network address of srv.Addr and then calls
// Receive to handle the incoming datagrams. If Addr is blank then
// DefaultMetricsAddr is used.
func (r *Server) ListenAndReceive() error {
	err := r.Listen()
	if err != nil {
		return err
	}

	return r.Receive(r.c)
}

func (r *Server) Close() error {
	return r.c.Close()
}

// How big to make the receive buffer. This default size is enough
// for a jumbo ethernet frame and the largest UDP packet possible.
const MaxPacket = 8096

// Receive accepts incoming datagrams on c and calls r.Handler.HandleMetric()
// for each line in the datagram that successfully parses in to a Metric
func (r *Server) Receive(c net.PacketConn) error {
	defer c.Close()

	pkt := make([]byte, MaxPacket)

	for {
		nbytes, _, err := c.ReadFrom(pkt)
		if err != nil {
			if strings.Contains(err.Error(), "closed") {
				return nil
			}

			log.Printf("Error reading: %s", err)
			continue
		}

		r.handlePacket(pkt[:nbytes])
	}

	panic("not reached")
}

// handleMessage handles the contents of a datagram and attempts to parse a Metric from each line
func (srv *Server) handlePacket(pkt []byte) error {
	metrics, err := parsePacket(pkt)
	if err != nil {
		return err
	}

	for _, m := range metrics {
		srv.Handler.HandleMetric(m)
	}

	return nil
}

var ErrInvalidFormat = errors.New("invalid format")

func parsePacket(pkt []byte) ([]*Metric, error) {
	var (
		metrics []*Metric
		single  []byte
	)

	for {
		nl := bytes.IndexRune(pkt, '\n')
		if nl == -1 {
			single = pkt
		} else {
			single = pkt[:nl]
		}

		metric, err := parseLine(single)
		if err != nil {
			return nil, err
		}

		metrics = append(metrics, metric)

		if nl == -1 {
			break
		}

		pkt = pkt[nl+1:]
		if len(pkt) == 0 {
			break
		}
	}

	return metrics, nil

	metric, err := parseLine(pkt)
	if err != nil {
		return nil, err
	}

	return []*Metric{metric}, nil
}

func parseLine(line []byte) (*Metric, error) {
	metric := &Metric{}

	bucket := bytes.IndexByte(line, ':')
	if bucket == -1 {
		return nil, ErrInvalidFormat
	}

	metric.Bucket = string(line[:bucket])

	rest := line[bucket+1:]

	valuePos := bytes.IndexByte(rest, '|')
	if valuePos == -1 {
		return nil, ErrInvalidFormat
	}

	value := string(rest[:valuePos])

	var err error

	metric.Value, err = strconv.ParseFloat(value, 64)
	if err != nil {
		return metric, fmt.Errorf("error converting metric value: %s", err)
	}

	rest = rest[valuePos+1:]

	sampleRate := float64(1)

	if atPos := bytes.IndexByte(rest, '@'); atPos != -1 {
		sampleRate, err = strconv.ParseFloat(string(rest[atPos+1:]), 10)
		if err != nil {
			return nil, err
		}

	}

	switch rest[0] {
	case 'm':
		metric.Type = TIMER
		metric.Value = metric.Value * (1.0 / sampleRate)
	case 'g':
		if value[0] == '+' || value[0] == '-' {
			metric.Type = GAUGE_DELTA
		} else {
			metric.Type = GAUGE
		}
	case 's':
		metric.Type = SET
	default:
		metric.Type = COUNTER
		metric.Value = metric.Value * (1.0 / sampleRate)
	}

	return metric, nil
}

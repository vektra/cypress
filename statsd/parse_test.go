package statsd

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vektra/neko"
)

func TestParse(t *testing.T) {
	n := neko.Start(t)

	n.It("can parse a counter", func() {
		var buf bytes.Buffer
		buf.WriteString("gorets:1|c")

		metric, err := parseLine(buf.Bytes())
		require.NoError(t, err)

		assert.Equal(t, metric.Type, COUNTER)
		assert.Equal(t, metric.Bucket, "gorets")
		assert.Equal(t, metric.Value, float64(1))
	})

	n.It("treats all unknown types as counters", func() {
		var buf bytes.Buffer
		buf.WriteString("gorets:1|x")

		metric, err := parseLine(buf.Bytes())
		require.NoError(t, err)

		assert.Equal(t, metric.Type, COUNTER)
		assert.Equal(t, metric.Bucket, "gorets")
		assert.Equal(t, metric.Value, float64(1))
	})

	n.It("can parse a counter with a sample rate", func() {
		var buf bytes.Buffer
		buf.WriteString("gorets:1|c|@0.1")

		metric, err := parseLine(buf.Bytes())
		require.NoError(t, err)

		assert.Equal(t, metric.Type, COUNTER)
		assert.Equal(t, metric.Bucket, "gorets")
		assert.Equal(t, metric.Value, float64(10))
	})

	n.It("can parse a gauge", func() {
		var buf bytes.Buffer
		buf.WriteString("gorets:83|g")

		metric, err := parseLine(buf.Bytes())
		require.NoError(t, err)

		assert.Equal(t, metric.Type, GAUGE)
		assert.Equal(t, metric.Bucket, "gorets")
		assert.Equal(t, metric.Value, float64(83))
	})

	n.It("can parse a gauge positive delta", func() {
		var buf bytes.Buffer
		buf.WriteString("gorets:+1|g")

		metric, err := parseLine(buf.Bytes())
		require.NoError(t, err)

		assert.Equal(t, metric.Type, GAUGE_DELTA)
		assert.Equal(t, metric.Bucket, "gorets")
		assert.Equal(t, metric.Value, float64(1))
	})

	n.It("can parse a gauge negative delta", func() {
		var buf bytes.Buffer
		buf.WriteString("gorets:-1|g")

		metric, err := parseLine(buf.Bytes())
		require.NoError(t, err)

		assert.Equal(t, metric.Type, GAUGE_DELTA)
		assert.Equal(t, metric.Bucket, "gorets")
		assert.Equal(t, metric.Value, float64(-1))
	})

	n.It("can parse a timiing event", func() {
		var buf bytes.Buffer

		buf.WriteString("gorets:83|ms")

		metric, err := parseLine(buf.Bytes())
		require.NoError(t, err)

		assert.Equal(t, metric.Type, TIMER)
		assert.Equal(t, metric.Bucket, "gorets")
		assert.Equal(t, metric.Value, float64(83))
	})

	n.It("can parse a set", func() {
		var buf bytes.Buffer

		buf.WriteString("gorets:3241|s")

		metric, err := parseLine(buf.Bytes())
		require.NoError(t, err)

		assert.Equal(t, metric.Type, SET)
		assert.Equal(t, metric.Bucket, "gorets")
		assert.Equal(t, metric.Value, float64(3241))
	})

	n.Meow()
}

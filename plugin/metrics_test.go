package plugin

import (
	"testing"
	"time"

	"github.com/rcrowley/go-metrics"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vektra/cypress"
	"github.com/vektra/neko"
)

func TestMetricsSink(t *testing.T) {
	n := neko.Start(t)

	var (
		ms *MetricSink
	)

	n.Setup(func() {
		ms = NewMetricSink()
	})

	n.It("saves metrics it sees to it's registry", func() {
		m := cypress.Metric()
		m.Add("name", "tests.run")
		m.Add("type", "counter")
		m.AddFloat("value", 10)

		err := ms.Receive(m)
		require.NoError(t, err)

		im, ok := ms.Registry.Get("tests.run").(metrics.Counter)
		require.True(t, ok)

		assert.Equal(t, 10, im.Count())
	})

	n.It("can handle a gauge", func() {
		m := cypress.Metric()
		m.Add("name", "tests.run")
		m.Add("type", "gauge")
		m.AddFloat("value", 10)

		err := ms.Receive(m)
		require.NoError(t, err)

		im, ok := ms.Registry.Get("tests.run").(metrics.GaugeFloat64)
		require.True(t, ok)

		assert.Equal(t, 10, im.Value())
	})

	n.It("can handle a timer", func() {
		m := cypress.Metric()
		m.Add("name", "tests.time")
		m.Add("type", "timer")
		m.AddDuration("value", 1234*time.Millisecond)

		err := ms.Receive(m)
		require.NoError(t, err)

		im, ok := ms.Registry.Get("tests.time").(metrics.Timer)
		require.True(t, ok)

		assert.Equal(t, 1234*time.Millisecond, im.Sum())
	})

	n.Meow()
}

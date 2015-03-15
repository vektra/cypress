package cypress

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vektra/neko"
)

func TestJSONMap(t *testing.T) {
	n := neko.Start(t)

	n.It("returns a Message in a map that converts to generic JSON", func() {
		m := Log()
		m.AddTag("region", "us-west-1")
		m.Add("hello", "world")
		m.Add("pid", 1)
		m.Add("rate", 3.3)
		m.Add("bytes", []byte("blah"))
		m.Add("awesome", true)
		m.Add("sucks", false)
		m.AddDuration("timing", 342*time.Millisecond)

		p := m.SimpleJSONMap()

		assert.Equal(t, "log", p["@type"])
		assert.Equal(t, m.Timestamp.Time().Format(time.RFC3339Nano), p["@timestamp"])
		tags := p["@tags"].(map[string]string)
		assert.Equal(t, "us-west-1", tags["region"])
		assert.Equal(t, "world", p["hello"])
		assert.Equal(t, 1, p["pid"])
		assert.Equal(t, 3.3, p["rate"])
		assert.Equal(t, map[string][]byte{"bytes": []byte("blah")}, p["bytes"])
		assert.Equal(t, true, p["awesome"])
		assert.Equal(t, false, p["sucks"])

		ivm := map[string]uint64{
			"seconds":     0,
			"nanoseconds": uint64(342 * time.Millisecond),
		}

		assert.Equal(t, ivm, p["timing"])
	})

	n.It("can create a new Message given the simple JSON format", func() {
		m := Log()
		m.AddTag("region", "us-west-1")
		m.Add("hello", "world")
		m.AddInt("pid", 1)
		m.Add("rate", 3.3)
		m.Add("awesome", true)
		m.Add("bytes", []byte("blah"))
		m.AddDuration("timing", 342*time.Millisecond)

		p := m.SimpleJSONMap()

		data, err := json.Marshal(p)
		require.NoError(t, err)

		m2, err := ParseSimpleJSON(data)
		require.NoError(t, err)

		assert.Equal(t, m.Timestamp, m2.Timestamp)
		assert.Equal(t, m.Type, m2.Type)
		assert.Equal(t, m.Tags, m2.Tags)

		str, ok := m2.GetString("hello")
		require.True(t, ok)

		assert.Equal(t, "world", str)

		i, ok := m2.GetInt("pid")
		require.True(t, ok)

		assert.Equal(t, 1, i)

		f, ok := m2.GetFloat("rate")
		require.True(t, ok)

		assert.Equal(t, 3.3, f)

		b, ok := m2.GetBool("awesome")
		require.True(t, ok)

		assert.Equal(t, true, b)

		bytes, ok := m2.GetBytes("bytes")
		require.True(t, ok)

		assert.Equal(t, []byte("blah"), bytes)

		iv, ok := m2.GetInterval("timing")
		require.True(t, ok)

		assert.Equal(t, 342*time.Millisecond, iv.Duration())
	})

	n.Meow()
}

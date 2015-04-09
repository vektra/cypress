package router

import (
	"strings"
	"testing"
	"time"

	"github.com/naoina/toml/ast"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vektra/cypress"
	"github.com/vektra/cypress/plugins/s3"
	"github.com/vektra/cypress/plugins/tcp"
	"github.com/vektra/neko"
)

func TestRouter(t *testing.T) {
	n := neko.Start(t)

	basicToml := `
[TCP]
address = ":8213"

[main.S3]
dir = "tmp/s3"
acl = "public"
region = "us-west-1"
access_key = "blah"

[route.Default]
generate = ["TCP"]
output = ["main"]
`

	n.It("loads configuration in via toml", func() {
		r := NewRouter()
		err := r.LoadConfig(strings.NewReader(basicToml))
		require.NoError(t, err)

		p1, ok := r.plugins["TCP"]
		require.True(t, ok)

		assert.Equal(t, "TCP", p1.Name)
		assert.Equal(t, "TCP", p1.Type)

		kv := p1.Config.Fields["address"].(*ast.KeyValue)
		assert.Equal(t, ":8213", kv.Value.(*ast.String).Value)

		p2, ok := r.plugins["main"]
		require.True(t, ok)

		assert.Equal(t, "main", p2.Name)
		assert.Equal(t, "S3", p2.Type)

		kv = p2.Config.Fields["access_key"].(*ast.KeyValue)
		assert.Equal(t, "blah", kv.Value.(*ast.String).Value)

		r1, ok := r.routes["Default"]
		require.True(t, ok)

		assert.Equal(t, true, r1.Enabled)
		assert.Equal(t, "Default", r1.Name)

		assert.Equal(t, []string{"TCP"}, r1.Generate)
		assert.Equal(t, []string{"main"}, r1.Output)
	})

	noRouteToml := `
[in.TCP]
address = ":8213"

[out.S3]
dir = "tmp/s3"
acl = "public"
region = "us-west-1"
access_key = "blah"
`

	n.It("creates a default route if none are given", func() {
		r := NewRouter()
		err := r.LoadConfig(strings.NewReader(noRouteToml))
		require.NoError(t, err)

		err = r.Open()
		require.NoError(t, err)

		defer r.Close()

		r1, ok := r.routes["Default"]
		require.True(t, ok)

		assert.Equal(t, true, r1.Enabled)
		assert.Equal(t, "Default", r1.Name)

		assert.Equal(t, []string{"in"}, r1.Generate)
		assert.Equal(t, []string{"out"}, r1.Output)
	})

	n.It("creates instances of the requested plugins", func() {
		r := NewRouter()
		err := r.LoadConfig(strings.NewReader(basicToml))
		require.NoError(t, err)

		err = r.Open()
		require.NoError(t, err)

		defer r.Close()

		tcp, ok := r.plugins["TCP"].Plugin.(*tcp.TCPPlugin)
		require.True(t, ok)

		assert.Equal(t, ":8213", tcp.Address)

		s3, ok := r.plugins["main"].Plugin.(*s3.S3Plugin)
		require.True(t, ok)

		assert.Equal(t, "blah", s3.AccessKey)
	})

	n.It("wires up generators to outputs per route", func() {
		testToml := `
[input.Test]

[output.Test]

[route.Default]
generate = ["input"]
output = ["output"]
`

		r := NewRouter()
		err := r.LoadConfig(strings.NewReader(testToml))
		require.NoError(t, err)

		err = r.Open()
		require.NoError(t, err)

		defer r.Close()

		r1, ok := r.routes["Default"]
		require.True(t, ok)

		in := r1.generators[0].(*cypress.TestPlugin)

		m := cypress.Log()
		m.Add("hello", "world")

		in.Messages <- m

		out := r1.receivers[0].(*cypress.TestPlugin)

		select {
		case m2 := <-out.Messages:
			assert.Equal(t, m, m2)
		case <-time.Tick(1 * time.Second):
			t.Fatal("message did not flow through the router")
		}
	})

	n.It("wires up filters in the routes", func() {
		testToml := `
[input.Test]

[output.Test]

[filt.Test]

[route.Default]
generate = ["input"]
filter = ["filt"]
output = ["output"]
`

		r := NewRouter()
		err := r.LoadConfig(strings.NewReader(testToml))
		require.NoError(t, err)

		err = r.Open()
		require.NoError(t, err)

		defer r.Close()

		r1, ok := r.routes["Default"]
		require.True(t, ok)

		filt := r1.filters[0].(*cypress.TestPlugin)

		filt.FilterFields = map[string]interface{}{
			"host": "filters.rock",
		}

		in := r1.generators[0].(*cypress.TestPlugin)

		m := cypress.Log()
		m.Add("hello", "world")

		in.Messages <- m

		out := r1.receivers[0].(*cypress.TestPlugin)

		select {
		case m2 := <-out.Messages:
			host, ok := m2.GetString("host")
			require.True(t, ok)

			assert.Equal(t, "filters.rock", host)
		case <-time.Tick(1 * time.Second):
			t.Fatal("message did not flow through the router")
		}
	})

	n.Meow()
}

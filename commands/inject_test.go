package commands

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/vektra/cypress"
	"github.com/vektra/neko"
)

func TestInject(t *testing.T) {
	n := neko.Start(t)

	var (
		mr  cypress.MockReceiver
		inj *Inject
		buf bytes.Buffer
	)

	n.CheckMock(&mr.Mock)

	n.Setup(func() {
		buf.Reset()
		inj = NewInject(&buf, &mr)
	})

	n.It("reads a message and sends it to a receiver from kv format", func() {
		m := cypress.Log()
		m.Add("hello", "world")

		str := m.KVString()

		buf.WriteString(str + "\n")

		mr.On("Receive", m).Return(nil)

		err := inj.Run()
		require.NoError(t, err)
	})

	n.It("reads a message and sends it to a receiver from json format", func() {
		m := cypress.Log()
		m.Add("hello", "world")

		err := json.NewEncoder(&buf).Encode(m)
		require.NoError(t, err)

		mr.On("Receive", m).Return(nil)

		err = inj.Run()
		require.NoError(t, err)
	})

	n.It("reads a message and sends it to a receiver from native format", func() {
		m := cypress.Log()
		m.Add("hello", "world")

		enc := cypress.NewEncoder(&buf)

		_, err := enc.Encode(m)

		mr.On("Receive", m).Return(nil)

		err = inj.Run()
		require.NoError(t, err)
	})
	n.Meow()
}

package plugin

import (
	"regexp"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/vektra/cypress"
	"github.com/vektra/neko"
)

func TestGrep(t *testing.T) {
	n := neko.Start(t)

	var mr cypress.MockReceiver

	n.CheckMock(&mr.Mock)

	n.It("filters messages that match the pattern through", func() {
		grep, err := NewGrep(&mr, "message", regexp.MustCompile("wo"))
		require.NoError(t, err)

		m := cypress.Log()
		m.Add("message", "hello world")

		mr.On("Read", m).Return(nil)

		m2 := cypress.Log()
		m2.Add("message", "hello people")

		err = grep.Read(m)
		require.NoError(t, err)

		err = grep.Read(m2)
		require.NoError(t, err)
	})

	n.It("can match a numeric field", func() {
		grep, err := NewGrep(&mr, "age", regexp.MustCompile("35"))
		require.NoError(t, err)

		m := cypress.Log()
		m.Add("age", 35)

		mr.On("Read", m).Return(nil)

		err = grep.Read(m)
		require.NoError(t, err)
	})

	n.Meow()
}

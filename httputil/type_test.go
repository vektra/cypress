package httputil

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/vektra/neko"
)

func TestContentType(t *testing.T) {
	n := neko.Start(t)

	n.It("picks a JSON streamer if the content type is json", func() {
		req, err := http.NewRequest("GET", "/", nil)
		require.NoError(t, err)

		req.Header.Add("Accept", "application/json")

		streamer := PickFromRequest(req, nil)
		require.NotNil(t, streamer)

		_, ok := streamer.(*JSONLogStreamer)
		require.True(t, ok)
	})

	n.It("picks a JSON streamer if there is no accept header", func() {
		req, err := http.NewRequest("GET", "/", nil)
		require.NoError(t, err)

		streamer := PickFromRequest(req, nil)
		require.NotNil(t, streamer)

		_, ok := streamer.(*JSONLogStreamer)
		require.True(t, ok)
	})

	n.It("picks a Protobuf streamer if the content type is vektralog", func() {
		req, err := http.NewRequest("GET", "/", nil)
		require.NoError(t, err)

		req.Header.Add("Accept", BinaryLogContentType)

		streamer := PickFromRequest(req, nil)
		require.NotNil(t, streamer)

		_, ok := streamer.(*ProtobufLogStreamer)
		require.True(t, ok)
	})

	n.It("picks a msgpack streamer if the content type is msgpack", func() {
		req, err := http.NewRequest("GET", "/", nil)
		require.NoError(t, err)

		req.Header.Add("Accept", "application/msgpack")

		streamer := PickFromRequest(req, nil)
		require.NotNil(t, streamer)

		_, ok := streamer.(*MsgpackLogStreamer)
		require.True(t, ok)
	})

	n.It("picks a key/value streamer if the content type is vektra-kvstream", func() {
		req, err := http.NewRequest("GET", "/", nil)
		require.NoError(t, err)

		req.Header.Add("Accept", TextLogContentType)

		streamer := PickFromRequest(req, nil)
		require.NotNil(t, streamer)

		_, ok := streamer.(*KVLogStreamer)
		require.True(t, ok)
	})

	n.Meow()
}

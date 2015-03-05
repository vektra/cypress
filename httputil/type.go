package httputil

import (
	"io"
	"net/http"

	"github.com/vektra/cypress"
)

type LogHandler interface {
	HandleMessage(m *cypress.Message) error
}

const BinaryLogContentType = "application/vnd.vektra-binarylog"
const TextLogContentType = "application/vnd.vektra-textlog"

func PickFromRequest(req *http.Request, w io.Writer) LogHandler {
	accept := req.Header.Get("Accept")

	if accept == "" {
		return JSONStreamer(w)
	}

	switch accept {
	case "application/json":
		return JSONStreamer(w)
	case "application/msgpack":
		return MsgpackStreamer(w)
	case BinaryLogContentType:
		return &ProtobufLogStreamer{w, asFlusher(w)}
	case TextLogContentType:
		return &KVLogStreamer{w: w, hf: asFlusher(w)}
	}

	return nil
}

func EstablishHandler(req *http.Request, w http.ResponseWriter) LogHandler {
	accept := req.Header.Get("Accept")

	if accept == "" {
		w.Header().Set("Content-Type", "application/json")
		return JSONStreamer(w)
	}

	switch accept {
	case "application/json":
		w.Header().Set("Content-Type", "application/json")
		return JSONStreamer(w)
	case "application/msgpack":
		w.Header().Set("Content-Type", "application/msgpack")
		return MsgpackStreamer(w)
	case BinaryLogContentType:
		w.Header().Set("Content-Type", BinaryLogContentType)
		return &ProtobufLogStreamer{w, asFlusher(w)}
	case TextLogContentType:
		w.Header().Set("Content-Type", TextLogContentType)
		return &KVLogStreamer{w: w, hf: asFlusher(w)}
	}

	return nil
}

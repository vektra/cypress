package httputil

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"io"
	"net/http"

	"github.com/ugorji/go/codec"
	"github.com/vektra/cypress"
)

type JSONLogStreamer struct {
	w   io.Writer
	hf  http.Flusher
	enc *json.Encoder
}

func asFlusher(w io.Writer) http.Flusher {
	if h, ok := w.(http.Flusher); ok {
		return h
	}

	return nil
}

func (h *JSONLogStreamer) HandleMessage(m *cypress.Message) error {
	err := h.enc.Encode(m)
	if err != nil {
		return err
	}

	if h.hf != nil {
		h.hf.Flush()
	}

	return nil
}

func JSONStreamer(w io.Writer) *JSONLogStreamer {
	return &JSONLogStreamer{w, asFlusher(w), json.NewEncoder(w)}
}

type ProtobufLogStreamer struct {
	w  io.Writer
	hf http.Flusher
}

func (h *ProtobufLogStreamer) HandleMessage(m *cypress.Message) error {
	data, err := m.Marshal()
	if err != nil {
		return err
	}

	size := uint32(len(data))

	var lenbuf [4]byte

	binary.BigEndian.PutUint32(lenbuf[:], size)

	_, err = h.w.Write(lenbuf[:])
	if err != nil {
		return err
	}

	_, err = h.w.Write(data)

	if h.hf != nil {
		h.hf.Flush()
	}

	return err
}

var msgpack codec.MsgpackHandle

type MsgpackLogStreamer struct {
	w   io.Writer
	hf  http.Flusher
	enc *codec.Encoder
}

func (h *MsgpackLogStreamer) HandleMessage(m *cypress.Message) error {
	err := h.enc.Encode(m)
	if err != nil {
		return err
	}

	if h.hf != nil {
		h.hf.Flush()
	}

	return nil
}

func MsgpackStreamer(w io.Writer) *MsgpackLogStreamer {
	return &MsgpackLogStreamer{w, asFlusher(w), codec.NewEncoder(w, &msgpack)}
}

type KVLogStreamer struct {
	w   io.Writer
	hf  http.Flusher
	buf bytes.Buffer
}

func (h *KVLogStreamer) HandleMessage(m *cypress.Message) error {
	h.buf.Reset()

	m.KVStringInto(&h.buf)

	h.buf.WriteString("\n")

	_, err := h.w.Write(h.buf.Bytes())

	if err != nil {
		return err
	}

	if h.hf != nil {
		h.hf.Flush()
	}

	return nil
}

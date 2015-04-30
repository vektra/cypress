package elasticsearch

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vektra/cypress"
	"github.com/vektra/neko"
)

func TestSend(t *testing.T) {
	n := neko.Start(t)

	n.It("store messages in elasticsearch", func() {
		var (
			body bytes.Buffer
			url  string
		)

		serv := http.Server{
			Addr: ":0",
			Handler: http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {

				url = req.URL.Path

				io.Copy(&body, req.Body)
				req.Body.Close()

				res.WriteHeader(200)
			}),
		}

		listener, err := net.Listen("tcp", ":0")
		require.NoError(t, err)

		defer listener.Close()

		port := listener.Addr().(*net.TCPAddr).Port

		go serv.Serve(listener)

		es := &Send{
			Host:  fmt.Sprintf("localhost:%d", port),
			Index: "cypress",
		}

		es.fixupHost()

		m := cypress.Log()
		m.Add("hello", "world")

		err = es.Receive(m)
		require.NoError(t, err)

		time.Sleep(100 * time.Millisecond)

		err = es.Close()
		require.NoError(t, err)

		l1 := `{"index":{"_index":"cypress","_type":"log","_timestamp":"%d"}}`
		l2, err := json.Marshal(m)

		assert.Equal(t,
			fmt.Sprintf(
				l1+"\n"+string(l2)+"\n",
				m.GetTimestamp().Time().UnixNano()/1e6,
			), body.String())

		assert.Equal(t, "/_bulk", url)
	})

	n.Meow()
}

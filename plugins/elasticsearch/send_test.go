package elasticsearch

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/vektra/cypress"
	"github.com/vektra/neko"
)

func TestSend(t *testing.T) {
	n := neko.Start(t)

	var conn MockConnection

	n.CheckMock(&conn.Mock)

	n.It("store messages in elasticsearch", func() {
		es := &Send{
			Host:  "http://localhost:9200",
			Index: "cypress",
			conn:  &conn,
		}

		m := cypress.Log()
		m.Add("hello", "world")

		data, err := json.Marshal(m)
		require.NoError(t, err)

		body := bytes.NewReader(data)
		req, err := http.NewRequest("POST", "http://localhost:9200/cypress/log", body)
		require.NoError(t, err)

		resp := &http.Response{
			Status:     "200 OK",
			StatusCode: 200,
			Body:       ioutil.NopCloser(strings.NewReader("")),
		}

		conn.On("Do", req).Return(resp, nil)

		err = es.Receive(m)
		require.NoError(t, err)
	})

	n.Meow()
}

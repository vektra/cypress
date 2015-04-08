package agent

import (
	"io/ioutil"
	"net"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/vektra/cypress"
	"github.com/vektra/neko"
)

func TestLocal(t *testing.T) {
	n := neko.Start(t)

	var mr cypress.MockReceiver

	n.CheckMock(&mr.Mock)

	tmpdir, err := ioutil.TempDir("", "log")
	require.NoError(t, err)

	defer os.RemoveAll(tmpdir)

	socket := filepath.Join(tmpdir, "cypress.sock")

	var lc *server

	n.It("reads logs off a unix socket", func() {
		lc = newServer(socket, &mr)

		var wg sync.WaitGroup

		m := cypress.Log()
		m.Add("hello", "tests")

		mr.On("Receive", m).Return(nil)

		wg.Add(1)

		go func() {
			defer wg.Done()
			err := lc.Start()
			require.NoError(t, err)
		}()

		time.Sleep(1 * time.Second)

		defer lc.Close()

		conn, err := net.Dial("unix", socket)
		require.NoError(t, err)

		defer conn.Close()

		enc := cypress.NewEncoder(conn)
		enc.Encode(m)

		time.Sleep(1 * time.Second)

		conn.Close()
		lc.Close()

		wg.Done()
	})

	n.Meow()
}

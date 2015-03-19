package plugin

import (
	"database/sql"
	"testing"
	"time"

	_ "github.com/lib/pq"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/vektra/cypress"
	"github.com/vektra/neko"
)

func TestPostgresql(t *testing.T) {

	n := neko.Start(t)

	var db MockDBInterface

	n.CheckMock(&db.Mock)

	n.It("sets up a db", func() {
		var p PostgreSQL
		p.Init(&db)

		db.On("Ping").Return(nil)
		db.On("Exec", cEnableHstore, []interface{}(nil)).Return(mock.Anything, nil)
		db.On("Exec", cCreateTable, []interface{}(nil)).Return(mock.Anything, nil)

		err := p.SetupDB()

		require.NoError(t, err)
	})

	n.It("receives a message", func() {
		var p PostgreSQL
		p.Init(&db)

		msg := cypress.Log()

		db.On("Exec", cAddRow, []interface{}{
			msg.GetTimestamp().Time().Format(time.RFC3339Nano),
			msg.Version,
			msg.Type,
			msg.SessionId,
			msg.HstoreAttributes(),
			msg.HstoreTags()}).Return(mock.Anything, nil)

		err := p.Receive(msg)

		require.NoError(t, err)
	})

	n.Meow()
}

func TestPostgreSQLOnline(t *testing.T) {
	n := neko.Start(t)

	// TODO: use ENV vars
	db, err := sql.Open("postgres", "user=jlsuttles dbname=vektra_test sslmode=disable")
	if err != nil {
		t.Skip()
	}

	n.It("sets up a db", func() {
		var p PostgreSQL
		p.Init(db)

		err := p.SetupDB()
		if err != nil {
			require.NoError(t, err)
		}

		require.NoError(t, err)
		// TODO: write sql stmt to check
	})

	n.It("receives a message", func() {
		var p PostgreSQL
		p.Init(db)

		err := p.SetupDB()
		require.NoError(t, err)

		msg := cypress.Log()
		msg.Add("message", "Hiiiii")
		msg.AddTag("key", "value")
		msg.AddTag("key2", "")

		err = p.Receive(msg)

		if err != nil {
			panic(err)
		}

		require.NoError(t, err)
		// TODO: write sql stmt to check
	})

	n.Meow()
}

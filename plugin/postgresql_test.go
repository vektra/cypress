package plugin

import (
	"database/sql"
	"testing"

	_ "github.com/lib/pq"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/vektra/cypress"
	"github.com/vektra/neko"
)

func TestPostgresql(t *testing.T) {

	n := neko.Start(t)

	var db MockDBInterface
	var stmt MockStmtInterface

	n.CheckMock(&db.Mock)
	n.CheckMock(&stmt.Mock)

	n.It("sets up a db", func() {
		var p PostgreSQL
		p.Init(&db)

		db.On("Ping").Return(nil)
		db.On("Exec", cCreateTable).Return(nil)

		err := p.SetupDB()

		require.NoError(t, err)
	})

	n.It("receives a message", func() {
		var p PostgreSQL
		p.Init(&db)

		msg := cypress.Log()
		msg.Add("message", "Hiiiii")

		stmt.On("Exec",
			[]interface{}{msg.Timestamp,
				msg.Version,
				msg.Type,
				msg.SessionId,
				msg.Attributes,
				msg.Tags}).Return(mock.Anything, nil)

		db.On("Prepare", cAddRow).Return(&stmt, nil)

		err := p.Receive(msg)

		require.NoError(t, err)
	})

	n.Meow()
}

func TestPostgresqlOnline(t *testing.T) {
	n := neko.Start(t)

	db, err := sql.Open("postgresql", "user=postgres dbname=cypress")
	if err != nil {
		t.Skip()
	}

	n.It("sets up a db", func() {
		var p PostgreSQL
		p.Init(db)

		err := p.SetupDB()

		require.NoError(t, err)
		// write sql stmt to check
	})

	n.Meow()
}

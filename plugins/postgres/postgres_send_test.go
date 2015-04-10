package postgres

import (
	"testing"
	"time"

	_ "github.com/lib/pq"
	"github.com/stretchr/testify/require"
	"github.com/vektra/cypress"
	"github.com/vektra/neko"
)

func TestPostgresql(t *testing.T) {

	n := neko.Start(t)

	var db MockDBInterface
	var res MockResultInterface

	n.CheckMock(&db.Mock)

	n.It("sets up a db", func() {
		var p Postgres
		p.Init(&db)

		db.On("Ping").Return(nil)
		db.On("Exec", cEnableHstore, []interface{}(nil)).Return(&res, nil)
		db.On("Exec", cCreateTable, []interface{}(nil)).Return(&res, nil)

		err := p.SetupDB()

		require.NoError(t, err)
	})

	n.It("receives a message", func() {
		var p Postgres
		p.Init(&db)

		msg := cypress.Log()

		db.On("Exec", cAddRow, []interface{}{
			msg.GetTimestamp().Time().Format(time.RFC3339Nano),
			msg.Version,
			msg.Type,
			msg.SessionId,
			msg.HstoreAttributes(),
			msg.HstoreTags()}).Return(&res, nil)

		err := p.Receive(msg)

		require.NoError(t, err)
	})

	n.Meow()
}

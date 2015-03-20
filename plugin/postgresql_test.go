package plugin

import (
	"database/sql"
	"fmt"
	"os"
	"testing"
	"time"

	_ "github.com/lib/pq"
	"github.com/stretchr/testify/require"
	"github.com/vektra/cypress"
	"github.com/vektra/neko"
)

const cUser = "TEST_POSTGRESQL_USER"
const cDBName = "TEST_POSTGRESQL_DB_NAME"

const cDropTable = `
DROP TABLE IF EXISTS cypress_messages
`

const cDropHstore = `
DROP EXTENSION IF EXISTS hstore CASCADE
`

const cCheckTableExists = `
SELECT 'public.cypress_messages'::regclass`

const cCheckHstoreExists = `
SELECT extname
FROM pg_extension
WHERE extname = 'hstore'`

func TestPostgresql(t *testing.T) {

	n := neko.Start(t)

	var db MockDBInterface
	var res MockResultInterface

	n.CheckMock(&db.Mock)

	n.It("sets up a db", func() {
		var p PostgreSQL
		p.Init(&db)

		db.On("Ping").Return(nil)
		db.On("Exec", cEnableHstore, []interface{}(nil)).Return(&res, nil)
		db.On("Exec", cCreateTable, []interface{}(nil)).Return(&res, nil)

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
			msg.HstoreTags()}).Return(&res, nil)

		err := p.Receive(msg)

		require.NoError(t, err)
	})

	n.Meow()
}

func TestPostgreSQLOnline(t *testing.T) {
	n := neko.Start(t)

	user := os.Getenv(cUser)
	if user == "" {
		t.Skipf("%s is not set.", cUser)
	}

	dbName := os.Getenv(cDBName)
	if dbName == "" {
		t.Skipf("%s is not set.", cDBName)
	}

	db, err := sql.Open("postgres",
		fmt.Sprintf("user=%s dbname=%s sslmode=disable", user, dbName))
	if err != nil {
		t.Skip(err)
	}

	err = db.Ping()
	if err != nil {
		t.Skipf("Could not connect to database: %s", err)
	}

	n.It("sets up a db", func() {
		var p PostgreSQL
		p.Init(db)

		var err error
		var hstore string

		db.Exec(cDropHstore)
		row := db.QueryRow(cCheckHstoreExists)
		row.Scan(&hstore)
		require.Equal(t, "", hstore)

		db.Exec(cDropTable)
		_, err = db.Exec(cCheckTableExists)
		require.Error(t, err)

		err = p.SetupDB()
		require.NoError(t, err)

		_, err = db.Exec(cCheckTableExists)
		require.NoError(t, err)

		row = db.QueryRow(cCheckHstoreExists)
		row.Scan(&hstore)
		require.Equal(t, "hstore", hstore)
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

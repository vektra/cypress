package postgres

import (
	"database/sql"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/vektra/cypress"
	"github.com/vektra/neko"
)

const cUser = "TEST_POSTGRES_USER"
const cDBName = "TEST_POSTGRES_DB_NAME"

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

const cCheckLatestMessage = `
SELECT timestamp, version, type, session_id, attributes, tags
FROM cypress_messages
ORDER BY timestamp
DESC
LIMIT 1`

func TestPostgresRecvBuildStmt(t *testing.T) {

	n := neko.Start(t)

	pr, _ := NewPostgresRecv(&Postgres{}, &Options{}, 1)

	n.It("empty options produces default query", func() {
		options := &Options{}
		expected := "SELECT timestamp, version, type, session_id, attributes, tags FROM cypress_messages WHERE timestamp IS NOT NULL ORDER BY timestamp DESC LIMIT 1"
		actual := pr.BuildStmt(options)

		require.Equal(t, expected, actual)
	})

	n.It("uses start and end options when present", func() {
		options := &Options{Start: "2014-01-01", End: "2015-01-01"}
		expected := "SELECT timestamp, version, type, session_id, attributes, tags FROM cypress_messages WHERE timestamp IS NOT NULL AND timestamp BETWEEN '2014-01-01' AND '2015-01-01' ORDER BY timestamp DESC LIMIT 1"
		actual := pr.BuildStmt(options)

		require.Equal(t, expected, actual)
	})

	n.It("uses start option when present", func() {
		options := &Options{Start: "2014-01-01"}
		expected := "SELECT timestamp, version, type, session_id, attributes, tags FROM cypress_messages WHERE timestamp IS NOT NULL AND timestamp >= '2014-01-01' ORDER BY timestamp DESC LIMIT 1"
		actual := pr.BuildStmt(options)

		require.Equal(t, expected, actual)
	})

	n.It("uses end option when present", func() {
		options := &Options{End: "2015-01-01"}
		expected := "SELECT timestamp, version, type, session_id, attributes, tags FROM cypress_messages WHERE timestamp IS NOT NULL AND timestamp <= '2015-01-01' ORDER BY timestamp DESC LIMIT 1"
		actual := pr.BuildStmt(options)

		require.Equal(t, expected, actual)
	})

	n.It("uses version option when present", func() {
		options := &Options{Version: 2}
		expected := "SELECT timestamp, version, type, session_id, attributes, tags FROM cypress_messages WHERE timestamp IS NOT NULL AND version = 2 ORDER BY timestamp DESC LIMIT 1"
		actual := pr.BuildStmt(options)

		require.Equal(t, expected, actual)
	})

	n.It("uses type option when present", func() {
		options := &Options{Type: 2}
		expected := "SELECT timestamp, version, type, session_id, attributes, tags FROM cypress_messages WHERE timestamp IS NOT NULL AND type = 2 ORDER BY timestamp DESC LIMIT 1"
		actual := pr.BuildStmt(options)

		require.Equal(t, expected, actual)
	})

	n.It("uses session id option when present", func() {
		options := &Options{SessionId: "aabbcc"}
		expected := "SELECT timestamp, version, type, session_id, attributes, tags FROM cypress_messages WHERE timestamp IS NOT NULL AND session_id = 'aabbcc' ORDER BY timestamp DESC LIMIT 1"
		actual := pr.BuildStmt(options)

		require.Equal(t, expected, actual)
	})

	n.It("uses attribute key and value options when present", func() {
		options := &Options{AttributeKey: "message", AttributeValue: "testing"}
		expected := "SELECT timestamp, version, type, session_id, attributes, tags FROM cypress_messages WHERE timestamp IS NOT NULL AND attributes->'message' = 'testing' ORDER BY timestamp DESC LIMIT 1"
		actual := pr.BuildStmt(options)

		require.Equal(t, expected, actual)
	})

	n.It("uses tag key and value options when present", func() {
		options := &Options{TagKey: "message", TagValue: "testing"}
		expected := "SELECT timestamp, version, type, session_id, attributes, tags FROM cypress_messages WHERE timestamp IS NOT NULL AND tags->'message' = 'testing' ORDER BY timestamp DESC LIMIT 1"
		actual := pr.BuildStmt(options)

		require.Equal(t, expected, actual)
	})

	n.It("uses asc order option when present", func() {
		options := &Options{Order: "ASC"}
		expected := "SELECT timestamp, version, type, session_id, attributes, tags FROM cypress_messages WHERE timestamp IS NOT NULL ORDER BY timestamp ASC LIMIT 1"
		actual := pr.BuildStmt(options)

		require.Equal(t, expected, actual)
	})

	n.It("uses limit option when present", func() {
		options := &Options{Limit: 100}
		expected := "SELECT timestamp, version, type, session_id, attributes, tags FROM cypress_messages WHERE timestamp IS NOT NULL ORDER BY timestamp DESC LIMIT 100"
		actual := pr.BuildStmt(options)

		require.Equal(t, expected, actual)
	})

	n.Meow()
}

func TestBufferMessages(t *testing.T) {
	n := neko.Start(t)

	defaults := &Options{}

	n.It("adds message to messages buffer", func() {
		pr, _ := NewPostgresRecv(&Postgres{}, defaults, 100)

		var messages []*cypress.Message
		expected := cypress.Log()
		messages = append(messages, expected)

		err := pr.BufferMessages(messages)
		require.NoError(t, err)

		actual := <-pr.MessageBuffer

		require.Equal(t, expected, actual)
	})

	n.It("sets start to be the timestamp from the added event", func() {
		pr, _ := NewPostgresRecv(&Postgres{}, defaults, 100)

		var messages []*cypress.Message
		expected := cypress.Log()
		messages = append(messages, expected)

		err := pr.BufferMessages(messages)
		require.NoError(t, err)

		require.Equal(t, expected.GetTimestamp().Time().Format(time.RFC3339), pr.Options.Start)
	})

	n.It("does not wait on full buffer", func() {
		pr, _ := NewPostgresRecv(&Postgres{}, defaults, 1)

		var messages []*cypress.Message
		expected := cypress.Log()
		extra := cypress.Log()
		messages = append(messages, expected)
		messages = append(messages, extra)

		err := pr.BufferMessages(messages)
		require.NoError(t, err)
	})

	n.Meow()
}

func TestPostgresOnline(t *testing.T) {
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
		var p Postgres
		p.Init(db)

		var hstore string
		var err error

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
		var p Postgres
		p.Init(db)

		var err error

		var (
			timestamp  time.Time
			version    int32
			msgType    uint32
			sessionId  string
			attributes string
			tags       string
		)

		err = p.SetupDB()
		require.NoError(t, err)

		msg := cypress.Log()
		msg.Add("message", "hiiiii")
		msg.AddTag("key", "value")
		sessionId = "123456"
		msg.SessionId = &sessionId

		err = p.Receive(msg)
		require.NoError(t, err)

		row := db.QueryRow(cCheckLatestMessage)
		err = row.Scan(&timestamp, &version, &msgType, &sessionId, &attributes, &tags)
		require.NoError(t, err)

		require.Equal(t, msg.GetTimestamp().Time().Format(time.RFC3339), timestamp.Format(time.RFC3339))
		require.Equal(t, msg.Version, version)
		require.Equal(t, *msg.Type, msgType)
		require.Equal(t, *msg.SessionId, sessionId)
		require.Equal(t, msg.HstoreAttributes(), attributes)
		require.Equal(t, msg.HstoreTags(), tags)

		options := &Options{
			Version:        msg.Version,
			Type:           *msg.Type,
			SessionId:      *msg.SessionId,
			AttributeKey:   "message",
			AttributeValue: "hiiiii",
			TagKey:         "key",
			TagValue:       "value",
			Order:          "ASC",
			Limit:          1,
		}

		pr, _ := NewPostgresRecv(&p, options, 1)

		msg, err = pr.Generate()
		if err != nil {
			panic(err)
		}

		require.Equal(t, msg.GetTimestamp().Time().Format(time.RFC3339), timestamp.Format(time.RFC3339))
		require.Equal(t, msg.Version, version)
		require.Equal(t, *msg.Type, msgType)
		require.Equal(t, *msg.SessionId, sessionId)

		tag, ok := msg.GetTag("key")

		require.True(t, ok)
		require.Equal(t, "value", tag)

		attribute, ok := msg.GetString("message")

		require.True(t, ok)
		require.Equal(t, "hiiiii", attribute)
	})

	n.Meow()
}

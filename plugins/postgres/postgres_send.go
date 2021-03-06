package postgres

import (
	"database/sql"
	"time"

	_ "github.com/lib/pq"
	"github.com/vektra/cypress"
)

const cEnableHstore = `
CREATE EXTENSION IF NOT EXISTS hstore
`

const cCreateTable = `
CREATE TABLE IF NOT EXISTS cypress_messages (
	timestamp TIMESTAMP,
	version INTEGER,
	type INTEGER,
	session_id TEXT,
	attributes HSTORE,
	tags HSTORE
)`

const cAddRow = `
INSERT INTO cypress_messages (
	timestamp,
	version,
	type,
	session_id,
	attributes,
	tags
) VALUES ($1, $2, $3, $4, $5, $6)`

type DBInterface interface {
	Ping() error
	Exec(query string, args ...interface{}) (sql.Result, error)
	Query(query string, args ...interface{}) (*sql.Rows, error)
	Close() error
}

type ResultInterface interface {
	LastInsertId() (int64, error)
	RowsAffected() (int64, error)
}

type Postgres struct {
	DB DBInterface
}

func (p *Postgres) Init(db DBInterface) {
	p.DB = db
}

func (p *Postgres) SetupDB() error {
	err := p.DB.Ping()
	if err != nil {
		return err
	}

	_, err = p.DB.Exec(cEnableHstore)
	if err != nil {
		return err
	}

	_, err = p.DB.Exec(cCreateTable)
	if err != nil {
		return err
	}

	// TODO: alter table if schema doesnt match ?

	return nil
}

func (p *Postgres) Receive(m *cypress.Message) error {
	_, err := p.DB.Exec(cAddRow,
		m.GetTimestamp().Time().Format(time.RFC3339Nano),
		m.Version,
		m.Type,
		m.SessionId,
		m.HstoreAttributes(),
		m.HstoreTags(),
	)
	if err != nil {
		return err
	}

	return nil
}

func (p *Postgres) Close() error {
	return p.DB.Close()
}

package plugin

import (
	"bytes"
	"database/sql"

	"github.com/vektra/cypress"
)

const cCreateTable = `
CREATE TABLE cypress_messages (
	timestamp 	timestamp
	version 		integer
	type 				integer
	session_id 	integer
	attributes 	hstore
	types 			hstore
)`

const cAddRow = `
INSERT INTO cypress_messages (
	timestamp,
	version,
	type,
	session_id,
	attributes,
	types,
) VALUES ($1, $2, $3, $4, $5, $6)`

type DBInterface interface {
	Ping() error
	Exec(string) error
	Prepare(string) (StmtInterface, error)
}

type StmtInterface interface {
	Exec(args ...interface{}) (sql.Result, error)
}

type PostgreSQL struct {
	Username string
	Password string
	Host     string
	Port     string
	DBName   string
	DB       DBInterface
}

func (p *PostgreSQL) dataSourceName() string {
	var buf bytes.Buffer
	buf.WriteString(p.Username)
	buf.WriteString(":")
	buf.WriteString(p.Password)
	buf.WriteString("@tcp(")
	buf.WriteString(p.Host)
	buf.WriteString(":")
	buf.WriteString(p.Port)
	buf.WriteString(")")
	buf.WriteString(p.DBName)
	return buf.String()
}

func (p *PostgreSQL) Init(db DBInterface) {
	p.DB = db
}

func (p *PostgreSQL) SetupDB() error {
	err := p.DB.Ping()
	if err != nil {
		return err
	}

	err = p.DB.Exec(cCreateTable)
	if err != nil {
		return err
	}

	return nil
}

func (p *PostgreSQL) Receive(m *cypress.Message) error {
	stmt, err := p.DB.Prepare(cAddRow)
	if err != nil {
		return err
	}

	_, err = stmt.Exec(
		m.Timestamp,
		m.Version,
		m.Type,
		m.SessionId,
		m.Attributes,
		m.Tags,
	)
	if err != nil {
		return err
	}

	return nil
}

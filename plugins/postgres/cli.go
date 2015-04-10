package postgres

import (
	"database/sql"
	"fmt"
	"os"

	"github.com/vektra/cypress"
	"github.com/vektra/cypress/cli/commands"
)

type Send struct {
	DBName      string `long:"dbname" description:"The name of the database to connect to"`
	User        string `long:"user" description:"The user to sign in as"`
	Password    string `long:"password" description:"The user's password"`
	Host        string `long:"host" description:"The host to connect to. Values that start with / are for unix domain sockets. (default is localhost)"`
	Port        string `long:"port" description:"The port to bind to. (default is 5432)"`
	SSLMode     string `long:"sslmode" description:"Whether or not to use SSL (default is require, this is not the default for libpq)"`
	Timeout     string `long:"timeout" description:"Maximum wait for connection, in seconds. Zero or not specified means wait indefinitely."`
	SSLCert     string `long:"sslcert" description:"Cert file location. The file must contain PEM encoded data."`
	SSLKey      string `long:"sslkey" description:"Key file location. The file must contain PEM encoded data."`
	SSLRootCert string `long:"sslrootcert" description:"The location of the root certificate file. The file must contain PEM encoded data."`
}

func (p *Send) Execute(args []string) error {
	db, err := sql.Open("postgres",
		fmt.Sprintf("dbname=%s user=%s password=%s host=%s port=%s sslmode=%s connect_timeout=%s sslcert=%s sslkey=%s sslrootcert=%s",
			p.DBName, p.User, p.Password, p.Host, p.Port, p.SSLMode, p.Timeout, p.SSLCert, p.SSLKey, p.SSLRootCert))
	if err != nil {
		return err
	}

	postgres := &Postgres{}
	postgres.Init(db)

	err = postgres.SetupDB()
	if err != nil {
		return err
	}

	dec, err := cypress.NewStreamDecoder(os.Stdin)
	if err != nil {
		return err
	}

	return cypress.Glue(dec, postgres)
}

type Recv struct {
	DBName      string `long:"dbname" description:"The name of the database to connect to"`
	User        string `long:"user" description:"The user to sign in as"`
	Password    string `long:"password" description:"The user's password"`
	Host        string `long:"host" description:"The host to connect to. Values that start with / are for unix domain sockets. (default is localhost)"`
	Port        string `long:"port" description:"The port to bind to. (default is 5432)"`
	SSLMode     string `long:"sslmode" description:"Whether or not to use SSL (default is require, this is not the default for libpq)"`
	Timeout     string `long:"timeout" description:"Maximum wait for connection, in seconds. Zero or not specified means wait indefinitely."`
	SSLCert     string `long:"sslcert" description:"Cert file location. The file must contain PEM encoded data."`
	SSLKey      string `long:"sslkey" description:"Key file location. The file must contain PEM encoded data."`
	SSLRootCert string `long:"sslrootcert" description:"The location of the root certificate file. The file must contain PEM encoded data."`

	Start          string
	End            string
	Version        int32
	Type           uint32
	SessionId      string
	AttributeKey   string
	AttributeValue string
	TagKey         string
	TagValue       string
	Order          string
	Limit          uint

	BufferSize int `long:"buffersize" default:"100"`
}

func (g *Recv) Execute(args []string) error {
	options := &Options{
		Start:     g.Start,
		End:       g.End,
		Version:   g.Version,
		Type:      g.Type,
		SessionId: g.SessionId,
		Order:     g.Order,
		Limit:     g.Limit,
	}

	db, err := sql.Open("postgres",
		fmt.Sprintf("dbname=%s user=%s password=%s host=%s port=%s sslmode=%s connect_timeout=%s sslcert=%s sslkey=%s sslrootcert=%s",
			g.DBName, g.User, g.Password, g.Host, g.Port, g.SSLMode, g.Timeout, g.SSLCert, g.SSLKey, g.SSLRootCert))
	if err != nil {
		return err
	}

	p := &Postgres{}
	p.Init(db)

	err = p.SetupDB()
	if err != nil {
		return err
	}

	postgres, err := NewPostgresRecv(p, options, g.BufferSize)
	if err != nil {
		return err
	}

	dec, err := cypress.NewStreamDecoder(os.Stdin)
	if err != nil {
		return err
	}

	return cypress.Glue(dec, postgres)

	receiver := cypress.NewStreamEncoder(os.Stdout)

	return cypress.Glue(postgres, receiver)
}

func init() {
	short := "Send messages to Postgres"
	long := "Given a stream on stdin, the postgres command will read those messages in and send them to Postgres."

	commands.Add("postgres:send", short, long, &Send{})

	short = "Get messages from Postgres"
	long = ""

	commands.Add("postgres:recv", short, long, &Recv{})
}

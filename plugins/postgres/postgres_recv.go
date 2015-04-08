package postgres

import (
	"bytes"
	"fmt"
	"log"
	"time"

	"github.com/vektra/cypress"
	"github.com/vektra/tai64n"
)

const (
	cStmt = `SELECT timestamp, version, type, session_id, attributes, tags FROM cypress_messages WHERE timestamp IS NOT NULL`

	cTimestampBtwnStmt = ` AND timestamp BETWEEN '%s' AND '%s'`
	cTimestampGTStmt   = ` AND timestamp >= '%s'`
	cTimestampLTStmt   = ` AND timestamp <= '%s'`
	cVersionStmt       = ` AND version = %d`
	cTypeStmt          = ` AND type = %d`
	cSessionStmt       = ` AND session_id = '%s'`
	cOrderAscStmt      = ` ORDER BY timestamp ASC`
	cOrderDescStmt     = ` ORDER BY timestamp DESC`
	cLimitStmt         = ` LIMIT %d`
)

type PostgresRecv struct {
	*Postgres
	Options       *Options
	BufferSize    int
	MessageBuffer chan *cypress.Message
}

type Options struct {
	Start     string
	End       string
	Version   int32
	Type      uint32
	SessionId string
	Order     string
	Limit     uint
}

func NewPostgresRecv(postgres *Postgres, options *Options, bufferSize int) (*PostgresRecv, error) {
	return &PostgresRecv{
		Postgres:      postgres,
		Options:       options,
		BufferSize:    bufferSize,
		MessageBuffer: make(chan *cypress.Message, bufferSize),
	}, nil
}

func (pr *PostgresRecv) BuildStmt(o *Options) string {
	var buf bytes.Buffer

	buf.WriteString(cStmt)

	if o.Start != "" && o.End != "" {
		buf.WriteString(fmt.Sprintf(cTimestampBtwnStmt, o.Start, o.End))
	} else {
		if o.Start != "" {
			buf.WriteString(fmt.Sprintf(cTimestampGTStmt, o.Start))
		}
		if o.End != "" {
			buf.WriteString(fmt.Sprintf(cTimestampLTStmt, o.End))
		}
	}

	if o.Version != 0 {
		buf.WriteString(fmt.Sprintf(cVersionStmt, o.Version))
	}

	if o.Type != 0 {
		buf.WriteString(fmt.Sprintf(cTypeStmt, o.Type))
	}

	if o.SessionId != "" {
		buf.WriteString(fmt.Sprintf(cSessionStmt, o.SessionId))
	}

	if o.Order == "asc" || o.Order == "ASC" {
		buf.WriteString(cOrderAscStmt)
	} else {
		buf.WriteString(cOrderDescStmt)
	}

	if o.Limit != 0 {
		buf.WriteString(fmt.Sprintf(cLimitStmt, o.Limit))
	} else {
		buf.WriteString(fmt.Sprintf(cLimitStmt, pr.BufferSize))
	}

	return buf.String()
}

func (pr *PostgresRecv) Search(o *Options) ([]*cypress.Message, error) {
	var (
		timestamp  time.Time
		version    int32
		msgType    uint32
		sessionId  string
		attributes string
		tags       string

		messages []*cypress.Message
	)

	stmt := pr.BuildStmt(o)

	rows, err := pr.DB.Query(stmt)
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	for rows.Next() {
		err := rows.Scan(&timestamp, &version, &msgType, &sessionId, &attributes, &tags)
		if err != nil {
			return nil, err
		}

		message := &cypress.Message{
			Timestamp: tai64n.FromTime(timestamp),
			Version:   version,
			Type:      &msgType,
			SessionId: &sessionId,
		}
		messages = append(messages, message)
	}

	err = rows.Err()
	if err != nil {
		return nil, err
	}

	return messages, nil
}

func (pr *PostgresRecv) BufferMessages(messages []*cypress.Message) error {
	for _, message := range messages {
		select {

		case pr.MessageBuffer <- message:
			// TODO: update something for paging

		default:
			break
		}
	}

	return nil
}

func (pr *PostgresRecv) Generate() (*cypress.Message, error) {
	select {

	case message := <-pr.MessageBuffer:
		return message, nil

	case <-time.After(time.Second * 1):
		return nil, nil

	default:
		messages, err := pr.Search(pr.Options)
		if err != nil {
			return nil, err
		}

		pr.BufferMessages(messages)

		return pr.Generate()
	}
}

func (pr *PostgresRecv) Close() error {
	close(pr.MessageBuffer)
	return nil
}

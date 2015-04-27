package elasticsearch

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"time"

	elastigo "github.com/mattbaird/elastigo/lib"
	"github.com/vektra/cypress"
	"github.com/vektra/errors"
)

type Connection interface {
	Do(*http.Request) (*http.Response, error)
}

type Send struct {
	conn     Connection
	Host     string `short:"H" long:"host" default:"localhost:9200" description:"Address of elasticsearch"`
	Index    string `short:"i" long:"index" description:"Store all messages in one index rather than date driven indexes"`
	Prefix   string `short:"p" long:"prefix" default:"cypress" description:"Prefix to apply to date driven indexes"`
	Logstash bool   `short:"l" long:"logstash" description:"Store messages like logstash does"`

	template string

	econn *elastigo.Conn
	bulk  *elastigo.BulkIndexer
}

func (s *Send) nextIndex() string {
	if s.Index != "" {
		return s.Index
	}

	return s.Prefix + "-" + time.Now().Format("2006.01.02")
}

func (s *Send) fixupHost() {
	if strings.HasPrefix(s.Host, "http://") ||
		strings.HasPrefix(s.Host, "https://") {
		return
	}

	if !strings.Contains(s.Host, ":") {
		s.Host = s.Host + ":9200"
	}

	s.Host = "http://" + s.Host
}

func (s *Send) connection() Connection {
	if s.conn == nil {
		return http.DefaultClient
	}

	return s.conn
}

// Check and write an index template for the indexes used
func (s *Send) SetupTemplate() error {
	req, err := http.NewRequest("GET", s.Host+"/_template/"+s.template, nil)
	if err != nil {
		return err
	}

	resp, err := s.connection().Do(req)
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	if resp.StatusCode < 300 {
		return nil
	}

	data, ok := Templates[s.template]
	if !ok {
		return fmt.Errorf("Unknown template: %s", s.template)
	}

	body := strings.NewReader(data)

	req, err = http.NewRequest("PUT", s.Host+"/_template/"+s.template, body)
	if err != nil {
		return err
	}

	resp, err = s.connection().Do(req)
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		str, _ := ioutil.ReadAll(resp.Body)

		return errors.Context(ErrFromElasticsearch, string(str))
	}

	return nil
}

var ErrFromElasticsearch = errors.New("elasticsearch reported an error")

// Write a Message to Elasticsearch
func (s *Send) Receive(m *cypress.Message) error {
	idx := s.nextIndex()

	if s.econn == nil {
		s.econn = elastigo.NewConn()
		s.bulk = s.econn.NewBulkIndexer(10)
	}

	t := m.GetTimestamp().Time()

	return s.bulk.Index(idx, m.StringType(), "", "", &t, m, false)
}

func (s *Send) Close() error {
	if s.bulk != nil {
		s.bulk.Flush()
	}

	return nil
}

// Check all the options and get ready to run.
func (s *Send) Init() error {
	if s.Logstash {
		s.template = "logstash"
		s.Index = ""
		s.Prefix = "logstash"
	} else {
		s.template = "cypress"
	}

	s.fixupHost()
	err := s.SetupTemplate()
	if err != nil {
		return err
	}

	return nil
}

// Called when used via the CLI
func (s *Send) Execute(args []string) error {
	dec, err := cypress.NewStreamDecoder(os.Stdin)
	if err != nil {
		return err
	}

	err = s.Init()
	if err != nil {
		return err
	}

	defer s.Close()

	return cypress.Glue(dec, s)
}

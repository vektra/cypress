package elasticsearch

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/vektra/cypress"
	"github.com/vektra/cypress/cli/commands"
	"github.com/vektra/errors"
)

type Connection interface {
	Do(*http.Request) (*http.Response, error)
}

type Store struct {
	conn     Connection
	Host     string `short:"H" long:"host" default:"localhost:9200" description:"Address of elasticsearch"`
	Index    string `short:"i" long:"index" description:"Store all messages in one index rather than date driven indexes"`
	Prefix   string `short:"p" long:"prefix" default:"cypress" description:"Prefix to apply to date driven indexes"`
	Logstash bool   `short:"l" long:"logstash" description:"Store messages like logstash does"`

	template string
}

func (s *Store) nextIndex() string {
	if s.Index != "" {
		return s.Index
	}

	return s.Prefix + "-" + time.Now().Format("2006.01.02")
}

func (s *Store) fixupHost() {
	if strings.HasPrefix(s.Host, "http://") ||
		strings.HasPrefix(s.Host, "https://") {
		return
	}

	if !strings.Contains(s.Host, ":") {
		s.Host = s.Host + ":9200"
	}

	s.Host = "http://" + s.Host
}

func (s *Store) connection() Connection {
	if s.conn == nil {
		return http.DefaultClient
	}

	return s.conn
}

// Check and write an index template for the indexes used
func (s *Store) SetupTemplate() error {
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
func (s *Store) Receive(m *cypress.Message) error {
	idx := s.nextIndex()
	url := s.Host + "/" + idx + "/" + m.StringType()

	data, err := json.Marshal(m)
	if err != nil {
		return err
	}

	body := bytes.NewReader(data)

	req, err := http.NewRequest("POST", url, body)
	if err != nil {
		return err
	}

	conn := s.conn
	if conn == nil {
		conn = http.DefaultClient
	}

	resp, err := conn.Do(req)
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		str, _ := ioutil.ReadAll(resp.Body)

		return errors.Context(ErrFromElasticsearch, string(str))
	}

	return err
}

// Check all the options and get ready to run.
func (s *Store) Init() error {
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
func (s *Store) Execute(args []string) error {
	dec, err := cypress.NewStreamDecoder(os.Stdin)
	if err != nil {
		return err
	}

	err = s.Init()
	if err != nil {
		return err
	}

	return cypress.Glue(dec, s)
}

// To fit the Generator interface
func (s *Store) Close() error {
	return nil
}

func init() {
	commands.Add("elasticsearch:send", "write messages to elasticsearch", "", &Store{})
}

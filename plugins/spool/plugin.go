package spool

import "github.com/vektra/cypress"

type SpoolPlugin struct {
	Directory string `description:"directory to read/write messages to"`
}

func (s *SpoolPlugin) Receiver() (cypress.Receiver, error) {
	return NewSpool(s.Directory)
}

func (s *SpoolPlugin) Generator() (cypress.Generator, error) {
	spool, err := NewSpool(s.Directory)
	if err != nil {
		return nil, err
	}

	return spool.Generator()
}

func init() {
	cypress.AddPlugin("Spool", func() cypress.Plugin { return &SpoolPlugin{} })
}

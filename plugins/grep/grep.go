package grep

import (
	"fmt"
	"regexp"

	"github.com/vektra/cypress"
)

type Grep struct {
	out    cypress.Receiver
	field  string
	regexp *regexp.Regexp
}

func NewGrep(out cypress.Receiver, field string, regexp *regexp.Regexp) (*Grep, error) {
	return &Grep{out, field, regexp}, nil
}

func (g *Grep) Filter(m *cypress.Message) (*cypress.Message, error) {
	if f, ok := m.Get(g.field); ok {
		var val string

		if s, ok := f.(string); ok {
			val = s
		} else {
			val = fmt.Sprintf("%s", f)
		}

		if g.regexp.MatchString(val) {
			return m, nil
		}
	}

	return nil, nil
}

func (g *Grep) Receive(m *cypress.Message) error {
	m2, err := g.Filter(m)
	if err != nil {
		return err
	}

	if m2 != nil {
		return g.out.Receive(m)
	}

	return nil
}

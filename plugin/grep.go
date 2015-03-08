package plugin

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

func (g *Grep) Read(m *cypress.Message) error {
	if f, ok := m.Get(g.field); ok {
		var val string

		if s, ok := f.(string); ok {
			val = s
		} else {
			val = fmt.Sprintf("%s", f)
		}

		if g.regexp.MatchString(val) {
			return g.out.Read(m)
		}
	}

	return nil
}

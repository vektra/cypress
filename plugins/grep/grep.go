package grep

import (
	"fmt"
	"regexp"

	"github.com/vektra/cypress"
	"github.com/vektra/cypress/cli/commands"
)

type Grep struct {
	Field   string `short:"f" long:"field" description:"field to match against"`
	Pattern string `short:"p" long:"pattern" description:"regexp pattern to match value against"`

	regexp *regexp.Regexp
}

func (g *Grep) Filter(m *cypress.Message) (*cypress.Message, error) {
	if g.regexp == nil {
		reg, err := regexp.Compile(g.Pattern)
		if err != nil {
			return nil, err
		}

		g.regexp = reg
	}

	if f, ok := m.Get(g.Field); ok {
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

func (g *Grep) Execute(args []string) error {
	return cypress.StandardStreamFilter(g)
}

func init() {
	commands.Add("grep", "filter messages using a regexp against a field", "", &Grep{})
	cypress.AddPlugin("grep", func() cypress.Plugin { return &Grep{} })
}

package json

import (
	"encoding/json"

	"github.com/vektra/cypress"
	"github.com/vektra/cypress/cli/commands"
)

type Filter struct {
	Field string `short:"f" long:"field" description:"json encoded field to expand"`
	Keep  bool   `short:"k" long:"keep" description:"keep the string field after expanding"`
}

func (f *Filter) Description() string {
	return `Parse a field as json and update the message with the resulting data.`
}

func (f *Filter) Filter(m *cypress.Message) (*cypress.Message, error) {
	str, ok := m.GetString(f.Field)
	if !ok {
		return m, nil
	}

	var data map[string]interface{}

	err := json.Unmarshal([]byte(str), &data)
	if err != nil {
		return m, nil
	}

	for k, v := range data {
		m.Add(k, v)
	}

	if !f.Keep {
		m.Remove(f.Field)
	}

	return m, nil
}

func (f *Filter) Filterer() (cypress.Filterer, error) {
	return f, nil
}

func (f *Filter) Execute(args []string) error {
	return cypress.StandardStreamFilter(f)
}

func init() {
	commands.Add("json", "expand json encoded string fields", "", &Filter{})
	cypress.AddPlugin("json", func() cypress.Plugin { return &Filter{} })
}

package file

import "github.com/vektra/cypress"
import "path/filepath"

type Plugin struct {
	Paths    []string `toml:"paths" description:"shell list (can contain *) of paths to watch lines for"`
	OffsetDB string   `toml:"offsetdb" description:"path to use to store file offsets"`
}

func (p *Plugin) Description() string {
	return `Generates messages from lines in files. Follows files as they change.`
}

func (p *Plugin) Generator() (cypress.Generator, error) {
	m := NewMonitor()

	var files []string

	for _, pat := range p.Paths {
		matches, err := filepath.Glob(pat)
		if err != nil {
			return nil, err
		}

		files = append(files, matches...)
	}

	if p.OffsetDB != "" {
		err := m.OpenOffsetDB(p.OffsetDB)
		if err != nil {
			return nil, err
		}
	}

	err := m.OpenFiles(false, files)
	if err != nil {
		return nil, err
	}

	return m.Generator()
}

func (p *Plugin) Receiver() (cypress.Receiver, error) {
	return nil, cypress.ErrNoReceiver
}

func init() {
	cypress.AddPlugin("file", func() cypress.Plugin { return &Plugin{} })
}

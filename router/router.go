package router

import (
	"io"
	"io/ioutil"
	"log"

	"github.com/naoina/toml"
	"github.com/naoina/toml/ast"
	"github.com/vektra/cypress"
	"github.com/vektra/errors"
)

type PluginDefinition struct {
	Name, Type string
	Config     *ast.Table
	Plugin     cypress.Plugin
}

type Route struct {
	Name     string
	Enabled  bool
	Generate []string
	Output   []string
	Filter   []string

	generators []cypress.Generator
	receivers  []cypress.Receiver
	filters    []cypress.Filterer
}

type Router struct {
	plugins map[string]*PluginDefinition
	routes  map[string]*Route
}

func NewRouter() *Router {
	return &Router{
		plugins: make(map[string]*PluginDefinition),
		routes:  make(map[string]*Route),
	}
}

func (r *Router) Routes() []*Route {
	var routes []*Route

	for _, route := range r.routes {
		routes = append(routes, route)
	}

	return routes
}

func (r *Router) LoadConfig(i io.Reader) error {
	data, err := ioutil.ReadAll(i)
	if err != nil {
		return err
	}

	ast, err := toml.Parse(data)
	if err != nil {
		return err
	}

	return r.loadPlugins(ast)
}

var ErrInvalidConfig = errors.New("invalid configuration")

func (r *Router) loadPlugins(top *ast.Table) error {
	for key, val := range top.Fields {
		typ := key

		switch sub := val.(type) {
		case *ast.Table:
			if len(sub.Fields) == 1 {
				for skey, sval := range sub.Fields {
					if stable, ok := sval.(*ast.Table); ok {
						typ = skey
						sub = stable
					}

					break
				}
			}

			if key == "route" {
				var route Route
				route.Enabled = true
				route.Name = typ

				err := toml.UnmarshalTable(sub, &route)
				if err != nil {
					return err
				}

				r.routes[typ] = &route
			} else {
				r.plugins[key] = &PluginDefinition{
					Name:   key,
					Type:   typ,
					Config: sub,
				}
			}
		default:
			return ErrInvalidConfig
		}
	}

	return nil
}

var (
	ErrUnknownPlugin    = errors.New("unknown plugin")
	ErrInvalidGenerator = errors.New("invalid/nil generator")
	ErrInvalidReceiver  = errors.New("invalid/nil receiver")
	ErrInvalidFilterer  = errors.New("invalid/nil filterer")
)

func (r *Router) Open() error {
	for name, def := range r.plugins {
		if def.Plugin == nil {
			plug, ok := cypress.FindPlugin(def.Type)
			if !ok {
				return errors.Subject(ErrUnknownPlugin, def.Type)
			}

			err := toml.UnmarshalTable(def.Config, plug)
			if err != nil {
				return errors.Subject(err, name)
			}

			def.Plugin = plug
		}
	}

	var err error

	if len(r.routes) == 0 {
		r.routes["Default"] = &Route{
			Name:     "Default",
			Enabled:  true,
			Generate: []string{"in"},
			Output:   []string{"out"},
		}
	}

	err = r.wireRoutes()
	if err != nil {
		return err
	}

	for _, route := range r.routes {
		if route.Enabled {
			go route.Flow()
		}
	}

	return nil
}

func (r *Router) wireRoutes() error {
	for _, route := range r.routes {
		if !route.Enabled {
			continue
		}

		for _, name := range route.Generate {
			def, ok := r.plugins[name]
			if !ok {
				return errors.Subject(ErrUnknownPlugin, name)
			}

			gp, ok := def.Plugin.(cypress.GeneratorPlugin)
			if !ok {
				return errors.Subject(ErrInvalidGenerator, name)
			}

			gen, err := gp.Generator()
			if err != nil {
				return errors.Subject(err, name)
			}

			if gen == nil {
				return errors.Subject(ErrInvalidGenerator, name)
			}

			route.generators = append(route.generators, gen)
		}

		for _, name := range route.Filter {
			def, ok := r.plugins[name]
			if !ok {
				return errors.Subject(ErrUnknownPlugin, name)
			}

			fp, ok := def.Plugin.(cypress.FiltererPlugin)
			if !ok {
				return errors.Subject(ErrInvalidFilterer, name)
			}

			filt, err := fp.Filterer()
			if err != nil {
				return errors.Subject(err, name)
			}

			if filt == nil {
				return errors.Subject(ErrInvalidFilterer, name)
			}

			route.filters = append(route.filters, filt)
		}

		for _, name := range route.Output {
			def, ok := r.plugins[name]
			if !ok {
				return errors.Subject(ErrUnknownPlugin, name)
			}

			rp, ok := def.Plugin.(cypress.ReceiverPlugin)
			if !ok {
				return errors.Subject(ErrInvalidGenerator, name)
			}

			recv, err := rp.Receiver()
			if err != nil {
				return errors.Subject(err, name)
			}

			if recv == nil {
				return errors.Subject(ErrInvalidReceiver, name)
			}

			route.receivers = append(route.receivers, recv)
		}
	}

	return nil
}

func (r *Route) Flow() {
	c := make(chan *cypress.Message, len(r.generators)*2)

	for _, g := range r.generators {
		go func(g cypress.Generator) {
			for {
				msg, err := g.Generate()
				if err != nil {
					log.Printf("Error generating messages: %s", err)
					return
				}

				c <- msg
			}
		}(g)
	}

	for msg := range c {
		for _, filt := range r.filters {
			msg, err := filt.Filter(msg)
			if err != nil {
				log.Printf("Error filtering message: %s", err)
				msg = nil
				break
			}

			if msg == nil {
				break
			}
		}

		if msg == nil {
			continue
		}

		for _, recv := range r.receivers {
			err := recv.Receive(msg)
			if err != nil {
				log.Printf("Error sending messages: %s", err)
			}
		}
	}
}

func (r *Router) Close() error {
	for _, route := range r.routes {
		for _, g := range route.generators {
			g.Close()
		}
	}

	return nil
}

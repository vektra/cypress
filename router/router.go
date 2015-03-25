package router

import (
	"io"
	"io/ioutil"

	"github.com/naoina/toml"
	"github.com/naoina/toml/ast"
	"github.com/vektra/cypress"
	_ "github.com/vektra/cypress/plugin"
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

	generators []cypress.Generator
	receivers  []cypress.Receiver
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

	for _, route := range r.routes {
		for _, name := range route.Generate {
			def, ok := r.plugins[name]
			if !ok {
				return errors.Subject(ErrUnknownPlugin, name)
			}

			gen, err := def.Plugin.Generator()
			if err != nil {
				return errors.Subject(err, name)
			}

			if gen == nil {
				return errors.Subject(ErrInvalidGenerator, name)
			}

			route.generators = append(route.generators, gen)
		}

		for _, name := range route.Output {
			def, ok := r.plugins[name]
			if !ok {
				return errors.Subject(ErrUnknownPlugin, name)
			}

			recv, err := def.Plugin.Receiver()
			if err != nil {
				return errors.Subject(err, name)
			}

			if recv == nil {
				return errors.Subject(ErrInvalidReceiver, name)
			}

			route.receivers = append(route.receivers, recv)
		}

		go route.Flow()
	}

	return nil
}

func (r *Route) Flow() {
	c := make(chan *cypress.Message, len(r.generators)*2)

	for _, g := range r.generators {
		go func(g cypress.Generator) {
			msg, err := g.Generate()
			if err != nil {
				return
			}

			c <- msg
		}(g)
	}

	for msg := range c {
		for _, recv := range r.receivers {
			recv.Receive(msg)
		}
	}
}

func (r *Router) Close() error {
	return nil
}

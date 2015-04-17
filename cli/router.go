package cli

import (
	"fmt"
	"os"
	"reflect"
	"strings"

	"github.com/vektra/cypress"
	"github.com/vektra/cypress/router"
)

type Router struct {
	ConfigFile string `short:"c" long:"config" description:"path to config file"`
	Available  bool   `short:"a" long:"available" description:"list all available plugins"`
}

type pluginDescription interface {
	Description() string
}

func (rt *Router) pluginType(v interface{}) string {
	var s []string

	if _, ok := v.(cypress.GeneratorPlugin); ok {
		s = append(s, "input")
	}

	if _, ok := v.(cypress.ReceiverPlugin); ok {
		s = append(s, "output")
	}

	if _, ok := v.(cypress.FiltererPlugin); ok {
		s = append(s, "filter")
	}

	return strings.Join(s, ", ")
}

func (rt *Router) showAvailable() error {
	for _, name := range cypress.AllPlugins() {
		pl, _ := cypress.FindPlugin(name)

		if pd, ok := pl.(pluginDescription); ok {
			desc := pd.Description()

			if desc == "<internal>" {
				continue
			}

			fmt.Printf("%s: %s\n  %s\n", name, rt.pluginType(pl), desc)
		} else {
			fmt.Printf("%s: %s\n", name, rt.pluginType(pl))
		}

		rt.showOptions(pl)

		fmt.Printf("\n")
	}

	return nil
}

func (rt *Router) showOptions(pl interface{}) {
	t := reflect.TypeOf(pl)

	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)

		if field.PkgPath != "" {
			continue
		}

		desc := field.Tag.Get("description")

		name := field.Tag.Get("toml")

		if name == "" {
			name = strings.ToLower(field.Name)
		}

		if desc != "" {
			fmt.Printf("  * %s: %s\n", name, desc)
		} else {
			fmt.Printf("  * %s\n", name)
		}
	}
}

func (rt *Router) Execute(args []string) error {
	if rt.Available {
		return rt.showAvailable()
	}

	r := router.NewRouter()

	f, err := os.Open(rt.ConfigFile)
	if err != nil {
		return err
	}

	defer f.Close()

	err = r.LoadConfig(f)
	if err != nil {
		return err
	}

	err = r.Open()
	if err != nil {
		return err
	}

	fmt.Printf("Router loaded and running\n%d routes active\n", len(r.Routes()))

	Lifecycle.OnShutdown(func() {
		r.Close()
	})

	select {}

	return nil
}

func init() {
	addCommand("router", "Route streams", "Route streams based on a config", &Router{})
}

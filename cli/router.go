package cli

import (
	"fmt"
	"os"

	"github.com/vektra/cypress/router"
)

type Router struct {
	ConfigFile string `short:"c" long:"config" description:"path to config file"`
}

func (rt *Router) Execute(args []string) error {
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

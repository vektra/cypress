package cypress

import (
	"os"
	"path/filepath"
	"sync"

	"github.com/mitchellh/go-homedir"
)

var globalConfig *Config

// Paths that can hold the global config
var GlobalConfigPaths []string

// Paths that, if they exist, are added to GlobalConfigPaths
var PotentialGlobalConfigPaths = []string{
	"/etc/cypress/config",
	"/var/lib/cypress/config",
}

// The path under a users home for the user config
var UserConfigPath = ".cypress/config"

var globalConfigLoaded sync.Once

// Whether or not to load the global Config from paths
var EmptyGlobalConfig bool

// Load and return the global Config
func GlobalConfig() *Config {
	globalConfigLoaded.Do(func() {
		if EmptyGlobalConfig {
			globalConfig = &Config{}
		} else {
			globalConfig = loadGlobalConfig()
		}
	})

	return globalConfig
}

func loadGlobalConfig() *Config {
	// Process the list in reverse order because we apply all files
	// to one config

	var cfg Config

	for i := len(GlobalConfigPaths) - 1; i >= 0; i-- {
		LoadMergedConfig(GlobalConfigPaths[i], &cfg)
	}

	return &cfg
}

func init() {
	dir, err := homedir.Dir()
	if err == nil && dir != "" {
		cfgdir := filepath.Join(dir, UserConfigPath)

		if _, err := os.Stat(cfgdir); err == nil {
			GlobalConfigPaths = append(GlobalConfigPaths, cfgdir)
		}
	}

	for _, path := range PotentialGlobalConfigPaths {
		if _, err := os.Stat(path); err == nil {
			GlobalConfigPaths = append(GlobalConfigPaths, path)
		}
	}
}

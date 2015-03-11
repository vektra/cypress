package cypress

import (
	"os"
	"path/filepath"
	"sync"

	"github.com/mitchellh/go-homedir"
)

var globalConfig *Config

var GlobalConfigPaths []string

var PotentialGlobalConfigPaths = []string{
	"/etc/cypress/config",
	"/var/lib/cypress/config",
}

var UserConfigPath = ".cypress/config"

var globalConfigLoaded sync.Once

func GlobalConfig() *Config {
	globalConfigLoaded.Do(func() {
		globalConfig = loadGlobalConfig()
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

package cypress

import (
	"os"
	"path/filepath"
	"sync"

	"github.com/mitchellh/go-homedir"
)

const CypressPathEnv = "CYPRESS_PATH"

var globalConfig *Config

// Paths that can contain any global cypress data
var GlobalPaths []string

// Paths that can hold the global config
var GlobalConfigPaths []string

// Paths that, if they exist, are added to GlobalConfigPaths
var PotentialGlobalPaths = []string{
	"/etc/cypress",
	"/var/lib/cypress",
}

var UserPath = ".cypress"

// The path under a users home for the user config
var UserConfigPath = UserPath + "/config"

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

// Retrieve the path for under the users .cypress directory
func UserFile(path string) (string, bool) {
	if HomeDir == "" {
		return "", false
	}

	fp := filepath.Join(HomeDir, UserPath, path)

	_, err := os.Stat(fp)
	if err != nil {
		return "", false
	}

	return fp, true
}

// Retrieve the path for under global cypress directories
func GlobalFile(path string) (string, bool) {
	for _, gp := range GlobalPaths {
		fp := filepath.Join(gp, path)

		_, err := os.Stat(fp)
		if err == nil {
			return fp, true
		}
	}

	return "", false
}

var HomeDir string

func init() {
	dir, err := homedir.Dir()
	if err == nil && dir != "" {
		HomeDir = dir

		os.Mkdir(filepath.Join(dir, UserPath), 0700)

		cfgdir := filepath.Join(dir, UserConfigPath)

		if _, err := os.Stat(cfgdir); err == nil {
			GlobalConfigPaths = append(GlobalConfigPaths, cfgdir)
		}
	}

	path := os.Getenv(CypressPathEnv)

	if path != "" {
		PotentialGlobalPaths = append(PotentialGlobalPaths, path)
	}

	for _, path := range PotentialGlobalPaths {
		if _, err := os.Stat(path); err == nil {
			GlobalPaths = append(GlobalPaths, path)

			cfg := filepath.Join(path, "config")
			if _, err := os.Stat(cfg); err == nil {
				GlobalConfigPaths = append(GlobalConfigPaths, cfg)
			}
		}
	}
}

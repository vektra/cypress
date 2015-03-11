package keystore

import (
	"log"
	"sync"
)

var (
	defaultUserKeys *UserKeys
	defaultKeys     Keys = defaultUserKeys
	setupKeys       sync.Once
)

func Default() Keys {
	setupKeys.Do(func() {
		err := defaultUserKeys.Setup()
		if err != nil {
			log.Fatal(err)
		}
	})

	return defaultKeys
}

func SetDefault(keys Keys) {
	defaultKeys = keys
}

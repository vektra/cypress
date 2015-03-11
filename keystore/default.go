package keystore

import (
	"log"
	"sync"
)

var defaultKeys *UserKeys

var setupKeys sync.Once

func Default() Keys {
	setupKeys.Do(func() {
		err := defaultKeys.Setup()
		if err != nil {
			log.Fatal(err)
		}
	})

	return defaultKeys
}

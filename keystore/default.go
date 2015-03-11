package keystore

var defaultKeys memoryKeys

func Default() Keys {
	return defaultKeys
}

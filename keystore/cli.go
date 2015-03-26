package keystore

import (
	"crypto/ecdsa"
	"fmt"

	"github.com/vektra/cypress/cli/commands"
)

type KeyGen struct {
	Name string `short:"n" long:"name" description:"Canonical name to include in the key"`

	Output string `short:"o" long:"output" description:"Where to write the key"	required:"true"`
}

func (k *KeyGen) Execute(args []string) error {
	return GenerateKey(k.Output, k.Name)
}

type KeyCheck struct {
	Args struct {
		File string
	} `positional-args:"yes" required:"true"`
}

func (k *KeyCheck) Execute(args []string) error {
	val, _, err := LoadPEM(k.Args.File)
	if err != nil {
		return err
	}

	switch key := val.(type) {
	case *ecdsa.PrivateKey:
		fmt.Printf("private key, %d bits\n", key.Params().BitSize)
	case *ecdsa.PublicKey:
		fmt.Printf("public key, %d bits\n", key.Params().BitSize)
	default:
		return fmt.Errorf("Unknown key type: %T\n", val)
	}

	return nil
}

func init() {
	commands.Add("key:gen", "generate a new crypto key", "", &KeyGen{})
	commands.Add("key:check", "inspect a crypto key", "", &KeyCheck{})
}

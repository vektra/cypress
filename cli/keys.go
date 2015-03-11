package cli

import (
	"crypto/ecdsa"
	"flag"
	"fmt"
	"log"

	"github.com/mitchellh/cli"
	"github.com/vektra/cypress/keystore"
)

type KeysCommand struct {
	Ui cli.Ui

	gen   string
	check string
}

func (k *KeysCommand) Synopsis() string {
	return "generate and manipulate keys"
}

func (k *KeysCommand) Help() string {
	return "get some help"
}

func (k *KeysCommand) Run(args []string) int {
	cmdFlags := flag.NewFlagSet("keys", flag.ContinueOnError)
	cmdFlags.StringVar(&k.gen, "gen", "", "")
	cmdFlags.StringVar(&k.check, "check", "", "")

	err := cmdFlags.Parse(args)
	if err != nil {
		return 1
	}

	if k.gen != "" {
		err = keystore.GenerateKey(k.gen)
		if err != nil {
			return 1
		}

		return 0
	}

	if k.check != "" {
		val, _, err := keystore.LoadPEM(k.check)
		if err != nil {
			log.Print(err)
			return 1
		}

		switch key := val.(type) {
		case *ecdsa.PrivateKey:
			fmt.Printf("private key, %d bits\n", key.Params().BitSize)
		case *ecdsa.PublicKey:
			fmt.Printf("public key, %d bits\n", key.Params().BitSize)
		default:
			fmt.Printf("Unknown key type: %T\n", val)
		}

		return 0
	}

	return 1
}

package main

import (
	"github.com/KelvinWu602/immutable-storage/blueprint"
)

func main() {
	// TODO
}

type Provider int

const (
	IPFS Provider = 0
)

func GetImmutableStorage(provider Provider, configFile string) blueprint.ImmutableStorage {
	// TODO
	return nil
}

package main

import (
	"crypto/rand"
	"time"

	"github.com/KelvinWu602/immutable-storage/blueprint"
	"github.com/KelvinWu602/immutable-storage/ipfs"
)

func main() {
	// TODO
	// experimental
	client := ipfs.NewIPFSClient(5 * time.Second)

	var key blueprint.Key
	rand.Read(key[:])
	client.GetDAGLinks("QmXjgEFmRVbikc12kwmehZn3f5w92y8qqVLQjrtx7N9AEq")
}

type Provider int

const (
	IPFS Provider = 0
)

func GetImmutableStorage(provider Provider, configFile string) blueprint.ImmutableStorage {
	// TODO
	return nil
}

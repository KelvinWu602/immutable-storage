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
	client.GetDAGLinks("QmaNN41g4oM6MDzZynrzpCwngBSe39poVE1Ket2rWznKZm")
}

type Provider int

const (
	IPFS Provider = 0
)

func GetImmutableStorage(provider Provider, configFile string) blueprint.ImmutableStorage {
	// TODO
	return nil
}

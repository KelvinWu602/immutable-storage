package main

import (
	"github.com/KelvinWu602/immutable-storage/blueprint"
)

func main() {
	// TODO
	// experimental
	// client := ipfs.NewIPFSClient(10 * time.Second)

	// res, err := client.CreateIPNSPointer("QmaNN41g4oM6MDzZynrzpCwngBSe39poVE1Ket2rWznKZm", "self")
	// if err != nil {
	// 	fmt.Println(err)
	// 	return
	// }
	// fmt.Println(res)
}

type Provider int

const (
	IPFS Provider = 0
)

func GetImmutableStorage(provider Provider, configFile string) blueprint.ImmutableStorage {
	// TODO
	return nil
}

package main

import (
	"github.com/KelvinWu602/immutable-storage/blueprint"
)

func main() {
	// TODO
	// experimental
	// client := ipfs.NewIPFSClient("127.0.0.1:5001", 10*time.Second)

	// err := client.CreateDirectory("/kelvin")
	// if err != nil {
	// 	fmt.Println(err)
	// 	if err.Error() == "files/mkdir: file already exists" {
	// 		fmt.Println("ok")
	// 	}
	// 	return
	// }
	//fmt.Println(res)
}

type Provider int

const (
	IPFS Provider = 0
)

func GetImmutableStorage(provider Provider, configFile string) blueprint.ImmutableStorage {
	// TODO
	return nil
}

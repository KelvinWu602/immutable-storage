package main

import (
	"github.com/KelvinWu602/immutable-storage/blueprint"
	"github.com/KelvinWu602/immutable-storage/ipfs"
	"github.com/KelvinWu602/immutable-storage/protos"
	"github.com/KelvinWu602/immutable-storage/server"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	"log"
	"net"
)

func main() {
	// TODO

	// experimental
	gs := grpc.NewServer()
	appServer := server.NewApplicationServer(ipfs.IPFS{})
	protos.RegisterImmutableStorageServer(gs, appServer)
	reflection.Register(gs)

	l, err := net.Listen("tcp", "127.0.0.1:3000")
	if err != nil {
		log.Fatal("Failed to create gRPC server on port 3000", "error", err)
	}
	gs.Serve(l)
}

type Provider int

const (
	IPFS Provider = 0
)

func GetImmutableStorage(provider Provider, configFile string) blueprint.ImmutableStorage {
	// TODO
	return nil
}

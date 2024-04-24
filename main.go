package main

import (
	"log"
	"net"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	"github.com/KelvinWu602/immutable-storage/protos"
	"github.com/KelvinWu602/immutable-storage/rmock"
	"github.com/KelvinWu602/immutable-storage/server"
)

func main() {
	gs := grpc.NewServer()
	log.Println("Setting up Immutable Storage...")
	redisImp := rmock.New()
	appServer := server.NewApplicationServer(redisImp)
	protos.RegisterImmutableStorageServer(gs, appServer)
	reflection.Register(gs)

	// TODO: Should listen to loopback address only, this is for testing purpose
	l, err := net.Listen("tcp", ":3100")
	// l, err := net.Listen("tcp", "127.0.0.1:3100")
	if err != nil {
		log.Fatal("Failed to create gRPC server on port 3100", "error", err)
	}

	log.Println("Setup completed. App Server running. Press Ctrl+C to stop.")
	// Blocking, waiting for gs.GracefulStop above.
	gs.Serve(l)

	log.Println("Exit successfully.")
}

// type Provider int

// const (
// 	IPFS Provider = 0
// )

// func GetImmutableStorage(provider Provider, configFile string) blueprint.ImmutableStorage {
// 	// TODO
// 	return nil
// }

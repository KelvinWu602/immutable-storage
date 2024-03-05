package main

import (
	"context"
	"log"
	"net"
	"os"
	"os/signal"
	"sync"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	"github.com/KelvinWu602/immutable-storage/ipfs"
	"github.com/KelvinWu602/immutable-storage/protos"
	"github.com/KelvinWu602/immutable-storage/server"
)

func main() {
	gs := grpc.NewServer()
	log.Println("Setting up Immutable Storage...")
	waitCtrlC, ctrlC := context.WithCancel(context.Background())
	ipfsImp, err := ipfs.New(waitCtrlC, "")
	appServer := server.NewApplicationServer(ipfsImp)
	protos.RegisterImmutableStorageServer(gs, appServer)
	reflection.Register(gs)

	// TODO: Should listen to loopback address only, this is for testing purpose
	l, err := net.Listen("tcp", ":3100")
	// l, err := net.Listen("tcp", "127.0.0.1:3100")
	if err != nil {
		log.Fatal("Failed to create gRPC server on port 3100", "error", err)
	}

	var waitSubgoroutines sync.WaitGroup
	waitSubgoroutines.Add(1)
	go func() {
		terminate := make(chan os.Signal, 1)
		signal.Notify(terminate, os.Interrupt)
		// wait for the SIGINT signal (Ctrl+C)
		<-terminate
		log.Println("Received interrupt signal. Please wait for the program to gracefully stop.")
		// Close AppServer first
		gs.GracefulStop()
		log.Println("Gracefully stopped App gRPC Server.")
		// Gracefully terminate all subgoroutines
		ctrlC()
		log.Println("Gracefully stopped all subgoroutines.")
		waitSubgoroutines.Done()
	}()

	log.Println("Setup completed. App Server running. Press Ctrl+C to stop.")
	// Blocking, waiting for gs.GracefulStop above.
	gs.Serve(l)
	waitSubgoroutines.Wait()

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

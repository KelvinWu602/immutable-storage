package server

import (
	"context"
	"errors"

	"github.com/KelvinWu602/immutable-storage/blueprint"
	"github.com/KelvinWu602/immutable-storage/protos"
)

type ApplicationServer struct {
	storage blueprint.ImmutableStorage
	protos.UnimplementedImmutableStorageServer
}

func NewApplicationServer(is blueprint.ImmutableStorage) ApplicationServer {
	return ApplicationServer{storage: is}
}

func (s ApplicationServer) Store(ctx context.Context, req *protos.StoreRequest) (*protos.StoreResponse, error) {
	if len(req.Key) != 48 {
		return &protos.StoreResponse{Success: false}, errors.New("Invalid Key")
	}
	key := [48]byte{}
	copy(key[:], req.Key)
	err := s.storage.Store(key, req.Content)
	if err == nil {
		return &protos.StoreResponse{Success: true}, nil
	} else {
		return &protos.StoreResponse{Success: false}, err
	}
}

func (s ApplicationServer) Read(ctx context.Context, req *protos.ReadRequest) (*protos.ReadResponse, error) {
	if len(req.Key) != 48 {
		return &protos.ReadResponse{Content: nil}, errors.New("Invalid Key")
	}
	key := [48]byte{}
	copy(key[:], req.Key)
	message, err := s.storage.Read(key)
	if err == nil {
		return &protos.ReadResponse{Content: message}, nil
	} else {
		return &protos.ReadResponse{Content: nil}, err
	}
}

func (s ApplicationServer) AvailableKeys(ctx context.Context, req *protos.AvailableKeysRequest) (*protos.AvailableKeysResponse, error) {
	return &protos.AvailableKeysResponse{Keys: nil}, nil
}

func (s ApplicationServer) mustEmbedUnimplementedImmutableStorageServer() {
}

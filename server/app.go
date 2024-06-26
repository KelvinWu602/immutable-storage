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

func (s ApplicationServer) Store(
	ctx context.Context,
	req *protos.StoreRequest,
) (*protos.StoreResponse, error) {
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

func (s ApplicationServer) Read(
	ctx context.Context,
	req *protos.ReadRequest,
) (*protos.ReadResponse, error) {
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

func (s ApplicationServer) AvailableKeys(
	ctx context.Context,
	req *protos.AvailableKeysRequest,
) (*protos.AvailableKeysResponse, error) {
	results := make([][]byte, 0)
	keys := s.storage.AvailableKeys()
	for _, key := range keys{
		key := key
		results = append(results, key[:])
	}
	return &protos.AvailableKeysResponse{Keys: results}, nil
}

func (s ApplicationServer) IsDiscovered(
	ctx context.Context,
	req *protos.IsDiscoveredRequest,
) (*protos.IsDiscoveredResponse, error) {
	return &protos.IsDiscoveredResponse{IsDiscovered: s.storage.IsDiscovered(blueprint.Key(req.Key))}, nil
}

func (s ApplicationServer) mustEmbedUnimplementedImmutableStorageServer() {
}

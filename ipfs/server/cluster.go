package server

import (
	"context"

	"github.com/KelvinWu602/immutable-storage/ipfs"
	"github.com/KelvinWu602/immutable-storage/ipfs/protos"
)

type ClusterServer struct {
	storage *ipfs.IPFS
	protos.UnimplementedImmutableStorageClusterServer
}

func NewClusterServer(ipfsimpl *ipfs.IPFS) ClusterServer {
	return ClusterServer{storage: ipfsimpl}
}

func (s ClusterServer) PropagateWrite(ctx context.Context, req *protos.PropagateWriteRequest) (*protos.PropagateWriteResponse, error) {
	return &protos.PropagateWriteResponse{}, nil
}

func (s ClusterServer) Sync(ctx context.Context, req *protos.SyncRequest) (*protos.SyncResponse, error) {
	return &protos.SyncResponse{
		Found:        false,
		CID:          "",
		MappingsIPNS: "",
	}, nil
}

func (s ClusterServer) GetNodetxtIPNS(ctx context.Context, req *protos.SyncRequest) (*protos.GetNodetxtIPNSResponse, error) {
	return &protos.GetNodetxtIPNSResponse{
		NodetxtIPNS: "",
	}, nil
}

func (s ClusterServer) mustEmbedUnimplementedImmutableStorageClusterServer() {

}

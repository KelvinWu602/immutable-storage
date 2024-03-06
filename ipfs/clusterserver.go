package ipfs

import (
	"context"

	"github.com/KelvinWu602/immutable-storage/blueprint"
	"github.com/KelvinWu602/immutable-storage/ipfs/protos"
)

type ClusterServer struct {
	storage *IPFS
	protos.UnimplementedImmutableStorageClusterServer
}

func NewClusterServer(ipfsimpl *IPFS) *ClusterServer {
	return &ClusterServer{storage: ipfsimpl}
}

func (s *ClusterServer) PropagateWrite(ctx context.Context, req *protos.PropagateWriteRequest) (*protos.PropagateWriteResponse, error) {
	return &protos.PropagateWriteResponse{}, s.storage.propagateWriteInternal(req.MappingsIPNS)
}

func (s *ClusterServer) Sync(ctx context.Context, req *protos.SyncRequest) (*protos.SyncResponse, error) {
	cid, source := s.storage.sync(blueprint.Key(req.Key))
	return &protos.SyncResponse{
		Found:        (cid != "" && source != ""),
		CID:          cid,
		MappingsIPNS: source,
	}, nil
}

func (s *ClusterServer) GetNodetxtIPNS(ctx context.Context, req *protos.GetNodetxtIPNSRequest) (*protos.GetNodetxtIPNSResponse, error) {
	return &protos.GetNodetxtIPNSResponse{
		NodetxtIPNS: s.storage.nodesIPNS,
	}, nil
}

func (s *ClusterServer) mustEmbedUnimplementedImmutableStorageClusterServer() {

}

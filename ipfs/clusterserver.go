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
	var cid, source string
	seen := s.storage.IsDiscovered(blueprint.Key(req.Key))
	if seen {
		cidProfile := s.storage.keyToCid[blueprint.Key(req.Key)]
		cid = cidProfile.cid
		source = cidProfile.source
	}
	return &protos.SyncResponse{
		Found:        seen,
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

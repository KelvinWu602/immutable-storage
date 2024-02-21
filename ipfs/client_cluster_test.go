package ipfs

import (
	"context"
	"log"
	"net"
	"testing"
	"time"

	"github.com/KelvinWu602/immutable-storage/ipfs/protos"
	"google.golang.org/grpc"
)

// MockClusterServer implements immutable-storage/protos/ImmutableStorageClusterServer interface
type MockClusterServer struct {
	protos.UnimplementedImmutableStorageClusterServer
}

func (s MockClusterServer) PropagateWrite(ctx context.Context, req *protos.PropagateWriteRequest) (*protos.PropagateWriteResponse, error) {
	return &protos.PropagateWriteResponse{}, nil
}

func (s MockClusterServer) Sync(ctx context.Context, req *protos.SyncRequest) (*protos.SyncResponse, error) {
	return &protos.SyncResponse{
		Found:        false,
		CID:          "CID",
		MappingsIPNS: "IPNS",
	}, nil
}

func (s MockClusterServer) GetNodetxtIPNS(ctx context.Context, req *protos.GetNodetxtIPNSRequest) (*protos.GetNodetxtIPNSResponse, error) {
	return &protos.GetNodetxtIPNSResponse{
		NodetxtIPNS: "IPNS",
	}, nil
}

func (s MockClusterServer) mustEmbedUnimplementedImmutableStorageClusterServer() {
}

func initImmutableStorageMockServer(t *testing.T) *grpc.Server {
	lis, err := net.Listen("tcp", "localhost:3101")
	if err != nil {
		t.Fatalf("error when init MockClusterServer: %s", err)
	}
	gs := grpc.NewServer()
	var mockServer MockClusterServer
	protos.RegisterImmutableStorageClusterServer(gs, mockServer)

	go func() {
		gs.Serve(lis)
	}()

	// Wait until the Mock Server is ready
	i := 1
	for conn, err := grpc.Dial("localhost:3101", grpc.WithInsecure(), grpc.WithBlock()); err != nil; {
		log.Printf("Test call %v %s\n", i, err)
		if conn != nil {
			conn.Close()
		}
		time.Sleep(1 * time.Second)
		i++
	}

	return gs
}

// func TestRequestSyncTimeoutJob(t *testing.T) {
// 	gs := initImmutableStorageMockServer(t)
// 	defer gs.GracefulStop()

// 	start := time.Now()
// 	requestSyncTimeoutJob(blueprint.Key([]byte("012345678901234567890123456789012345678901234567")), []string{"127.0.0.1", "127.0.0.1", "127.0.0.1", "127.0.0.1"}, 3*time.Second)
// 	end := time.Now()
// 	t.Logf("%v", end.Sub(start))
// }

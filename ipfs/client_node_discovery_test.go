package ipfs

import (
	"context"
	"log"
	"net"
	"testing"
	"time"

	"github.com/KelvinWu602/node-discovery/protos"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
)

// MockNodeDiscoveryServer implements node-discovery/protos/NodeDiscoveryServer interface
type MockNodeDiscoveryServer struct {
	protos.UnimplementedNodeDiscoveryServer
}

func (s MockNodeDiscoveryServer) JoinCluster(ctx context.Context, req *protos.JoinClusterRequest) (*protos.JoinClusterResponse, error) {
	return &protos.JoinClusterResponse{}, nil
}

func (s MockNodeDiscoveryServer) LeaveCluster(ctx context.Context, req *protos.LeaveClusterRequest) (*protos.LeaveClusterResponse, error) {
	return &protos.LeaveClusterResponse{}, nil
}

func (s MockNodeDiscoveryServer) GetMembers(ctx context.Context, req *protos.GetMembersRequest) (*protos.GetMembersReponse, error) {
	return &protos.GetMembersReponse{Member: []string{"10.0.0.1", "10.0.0.2", "10.0.0.3"}}, nil
}

func (s MockNodeDiscoveryServer) mustEmbedUnimplementedNodeDiscoveryServer() {
}

func initNodeDiscoveryMockServer(t *testing.T) *grpc.Server {
	lis, err := net.Listen("tcp", "localhost:3200")
	if err != nil {
		t.Fatalf("error when init MockNodeDiscoveryServer: %s", err)
	}
	gs := grpc.NewServer()
	var mockServer MockNodeDiscoveryServer
	protos.RegisterNodeDiscoveryServer(gs, mockServer)

	go func() {
		gs.Serve(lis)
	}()

	// Wait until the Mock Server is ready
	i := 1
	for conn, err := grpc.Dial("localhost:3200", grpc.WithInsecure()); err != nil; {
		log.Printf("Test call %v %s\n", i, err)
		if conn != nil {
			conn.Close()
		}
		time.Sleep(1 * time.Second)
		i++
	}

	return gs
}

func TestNodeDiscoveryClientCreationDestroy(t *testing.T) {
	gs := initNodeDiscoveryMockServer(t)
	defer gs.GracefulStop()

	nd, err := newNodeDiscoveryClient("localhost:3200", 1*time.Second)
	defer func() {
		if err := nd.closeNodeDiscoveryClient(); err != nil {
			t.Fatalf("error when called closeNodeDiscoveryClient: %s", err)
		}
	}()
	if err != nil {
		t.Fatalf("error when called newNodeDiscoveryClient: %s", err)
	}
	if nd.timeout != time.Second {
		t.Fatalf("expected: %v, actual: %v", time.Second, nd.timeout)
	}
}

func TestGetMembers(t *testing.T) {
	gs := initNodeDiscoveryMockServer(t)
	defer gs.GracefulStop()

	nd, err := newNodeDiscoveryClient("localhost:3200", 1*time.Second)
	defer func() {
		if err := nd.closeNodeDiscoveryClient(); err != nil {
			t.Fatalf("error when called closeNodeDiscoveryClient: %s", err)
		}
	}()
	if err != nil {
		t.Fatalf("error when called newNodeDiscoveryClient: %s", err)
	}

	members, err := nd.getMembers()
	if err != nil {
		t.Fatalf("error when called getMembers: %s", err)
	}
	assert := assert.New(t)
	assert.Equal("10.0.0.1", members[0], "1st member should be 10.0.0.1")
	assert.Equal("10.0.0.2", members[1], "2nd member should be 10.0.0.2")
	assert.Equal("10.0.0.3", members[2], "3rd member should be 10.0.0.3")
}

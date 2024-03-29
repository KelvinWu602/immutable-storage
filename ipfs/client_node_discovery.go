package ipfs

import (
	"context"
	"log"
	"math/rand"
	"time"

	"github.com/KelvinWu602/node-discovery/protos"
	"google.golang.org/grpc"
)

type nodeDiscoveryClient struct {
	conn    *grpc.ClientConn
	client  protos.NodeDiscoveryClient
	timeout time.Duration
}

func newNodeDiscoveryClient(addr string, timeout time.Duration) (*nodeDiscoveryClient, error) {
	conn, err := grpc.Dial(addr, grpc.WithInsecure(), grpc.WithBlock())
	if err != nil {
		log.Println(err)
		return nil, err
	}
	client := protos.NewNodeDiscoveryClient(conn)
	return &nodeDiscoveryClient{conn, client, timeout}, nil
}

func (nd *nodeDiscoveryClient) closeNodeDiscoveryClient() error {
	if err := nd.conn.Close(); err != nil {
		log.Println(err)
		return err
	}
	return nil
}

func (nd *nodeDiscoveryClient) getMembers() ([]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), nd.timeout)
	defer cancel()
	res, err := nd.client.GetMembers(ctx, &protos.GetMembersRequest{})
	if err != nil {
		log.Println(err)
		return nil, err
	}
	return res.Member, nil
}

func (nd *nodeDiscoveryClient) getNMembers(n int) ([]string, error) {
	members, err := nd.getMembers()
	if err != nil {
		log.Println(err)
		return nil, err
	}
	if len(members) <= n {
		return members, nil
	}
	subset := make([]string, n)
	choice := rand.Perm(len(members))[:n]
	for i := 0; i < n; i++ {
		subset[i] = members[choice[i]]
	}
	return subset, nil
}

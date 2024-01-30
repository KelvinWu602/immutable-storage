package ipfs

import (
	"context"
	"log"
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
	conn, err := grpc.Dial(addr, grpc.WithInsecure())
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

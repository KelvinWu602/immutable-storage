package ipfs

import (
	"context"
	"log"
	"time"

	"github.com/KelvinWu602/immutable-storage/blueprint"
	"github.com/KelvinWu602/immutable-storage/ipfs/protos"
	"google.golang.org/grpc"
)

type clusterClient struct {
	conn    *grpc.ClientConn
	client  protos.ImmutableStorageClusterClient
	timeout time.Duration
}

func newClusterClient(addr string, timeout time.Duration) (*clusterClient, error) {
	conn, err := grpc.Dial(addr, grpc.WithInsecure())
	if err != nil {
		log.Println(err)
		return nil, err
	}
	client := protos.NewImmutableStorageClusterClient(conn)
	return &clusterClient{conn, client, timeout}, nil
}

func (cls *clusterClient) closeClusterClient() error {
	if err := cls.conn.Close(); err != nil {
		log.Println(err)
		return err
	}
	return nil
}

func (cls *clusterClient) propagateWrite(mappingIPNS string) error {
	ctx, cancel := context.WithTimeout(context.Background(), cls.timeout)
	defer cancel()
	_, err := cls.client.PropagateWrite(ctx, &protos.PropagateWriteRequest{MappingsIPNS: mappingIPNS})
	if err != nil {
		log.Println(err)
		return err
	}
	return nil
}

func (cls *clusterClient) sync(key blueprint.Key) (found bool, cid string, mappingsIPNS string, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), cls.timeout)
	defer cancel()
	res, err := cls.client.Sync(ctx, &protos.SyncRequest{Key: key[:]})
	if err != nil {
		log.Println(err)
		return false, "", "", err
	}
	return res.Found, res.CID, res.MappingsIPNS, nil
}

func (cls *clusterClient) getNodetxtIPNS() (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), cls.timeout)
	defer cancel()
	res, err := cls.client.GetNodetxtIPNS(ctx, &protos.GetNodetxtIPNSRequest{})
	if err != nil {
		log.Println(err)
		return "", err
	}
	return res.NodetxtIPNS, nil
}

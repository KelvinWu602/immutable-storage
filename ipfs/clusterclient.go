package ipfs

import (
	"context"
	"time"

	"github.com/KelvinWu602/immutable-storage/ipfs/protos"
	"google.golang.org/grpc"
)

func createClusterClient(addr string, timeout time.Duration) (protos.ImmutableStorageClusterClient, *context.Context, *context.CancelFunc, error) {
	var opts []grpc.DialOption
	conn, err := grpc.Dial(addr, opts...)
	if err != nil {
		return nil, nil, nil, err
	}
	defer conn.Close()
	grpcclient := protos.NewImmutableStorageClusterClient(conn)
	// Contact the server and print out its response.
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	return grpcclient, &ctx, &cancel, nil
}

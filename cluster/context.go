package cluster

import (
	"context"

	"github.com/rancher/go-rancher/v3"
)

type keyType string

var (
	clusterKey = keyType("cluster")
)

func GetCluster(ctx context.Context) *client.Cluster {
	cluster, _ := ctx.Value(clusterKey).(*client.Cluster)
	return cluster
}

func StoreCluster(ctx context.Context, cluster *client.Cluster) context.Context {
	return context.WithValue(ctx, clusterKey, cluster)
}

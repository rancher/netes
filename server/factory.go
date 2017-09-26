package server

import (
	"net/http"

	"github.com/docker/docker/pkg/locker"
	"github.com/rancher/go-rancher/v3"
	"github.com/rancher/netes/cluster"
	"github.com/rancher/netes/server/embedded"
	"github.com/rancher/netes/types"
	"golang.org/x/sync/syncmap"
)

type Factory struct {
	clusterLookup *cluster.Lookup
	clusters      syncmap.Map
	config        *types.GlobalConfig
	serverLock    *locker.Locker
	servers       syncmap.Map
}

func NewFactory(config *types.GlobalConfig) *Factory {
	return &Factory{
		serverLock:    locker.New(),
		config:        config,
		clusterLookup: config.Lookup,
	}
}

func (s *Factory) lookupCluster(clusterID string) (*client.Cluster, http.Handler) {
	server, ok := s.servers.Load(clusterID)
	if ok {
		if cluster, ok := s.clusters.Load(clusterID); ok {
			return cluster.(*client.Cluster), server.(Server).Handler()
		}
	}

	return nil, nil
}

func (s *Factory) Get(req *http.Request) (*client.Cluster, http.Handler, error) {
	clusterID := cluster.GetClusterID(req)
	cluster, handler := s.lookupCluster(clusterID)
	if cluster != nil {
		return cluster, handler, nil
	}

	s.serverLock.Lock("cluster." + clusterID)
	defer s.serverLock.Unlock("cluster." + clusterID)

	cluster, handler = s.lookupCluster(clusterID)
	if cluster != nil {
		return cluster, handler, nil
	}

	cluster, err := s.clusterLookup.Lookup(req)
	if err != nil || cluster == nil {
		return nil, nil, err
	}

	if cluster.K8sServerConfig == nil {
		cluster.K8sServerConfig = &client.K8sServerConfig{}
	}

	var server interface{}
	server, err = s.newServer(cluster)
	if err != nil || server == nil {
		return nil, nil, err
	}

	server, _ = s.servers.LoadOrStore(cluster.Id, server)
	s.clusters.LoadOrStore(cluster.Id, cluster)

	return cluster, server.(Server).Handler(), nil
}

func (s *Factory) newServer(c *client.Cluster) (Server, error) {
	if c.Embedded {
		return embedded.New(s.config, c, s.config.Lookup)
	}

	return nil, nil
}

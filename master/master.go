package master

import (
	"fmt"
	"net/http"

	"github.com/rancher/netes/cluster"
	"github.com/rancher/netes/router"
	"github.com/rancher/netes/server"
	"github.com/rancher/netes/types"
	"k8s.io/kubernetes/pkg/capabilities"
)

func New(c *types.GlobalConfig) *Master {
	return &Master{
		config: c,
	}
}

type Master struct {
	config        *types.GlobalConfig
	serverFactory *server.Factory
}

func (m *Master) Run() error {
	capabilities.Initialize(capabilities.Capabilities{
		AllowPrivileged: true,
		PrivilegedSources: capabilities.PrivilegedSources{
			HostNetworkSources: []string{},
			HostPIDSources:     []string{},
			HostIPCSources:     []string{},
		},
		PerConnectionBandwidthLimitBytesPerSec: 0,
	})

	if m.config.Lookup == nil {
		m.config.Lookup = cluster.NewLookup(m.config.CattleURL + "/clusters")
	}

	m.serverFactory = server.NewFactory(m.config)
	r := router.New(m.config)

	fmt.Println("Listening on", m.config.ListenAddr)
	return http.ListenAndServe(m.config.ListenAddr, r)
}

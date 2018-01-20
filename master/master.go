package master

import (
	"net/http"

	"github.com/rancher/netes/router"
	"github.com/rancher/netes/types"
	"k8s.io/kubernetes/pkg/capabilities"
)

func New(c *types.GlobalConfig) http.Handler {
	capabilities.Initialize(capabilities.Capabilities{
		AllowPrivileged: true,
		PrivilegedSources: capabilities.PrivilegedSources{
			HostNetworkSources: []string{},
			HostPIDSources:     []string{},
			HostIPCSources:     []string{},
		},
		PerConnectionBandwidthLimitBytesPerSec: 0,
	})

	return router.New(c)
}

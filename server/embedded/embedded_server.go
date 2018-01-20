package embedded

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/go-openapi/spec"
	"github.com/pkg/errors"
	"github.com/rancher/netes/clients"
	"github.com/rancher/netes/cluster"
	"github.com/rancher/netes/proxy"
	"github.com/rancher/netes/server/admission"
	"github.com/rancher/netes/store"
	"github.com/rancher/netes/types"
	"github.com/rancher/rancher/k8s/apiserver/auth"
	"github.com/rancher/types/apis/management.cattle.io/v3"
	utilnet "k8s.io/apimachinery/pkg/util/net"
	"k8s.io/apimachinery/pkg/util/sets"
	genericapiserver "k8s.io/apiserver/pkg/server"
	"k8s.io/apiserver/pkg/server/filters"
	"k8s.io/apiserver/pkg/server/storage"
	"k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/generated/openapi"
	kubeletclient "k8s.io/kubernetes/pkg/kubelet/client"
	"k8s.io/kubernetes/pkg/master"
	"k8s.io/kubernetes/pkg/master/ports"
	"k8s.io/kubernetes/pkg/version"
)

type Server struct {
	master  *master.Master
	cluster *v3.Cluster
	cancel  context.CancelFunc
}

func (e *Server) Close() {
	e.cancel()
}

func (e *Server) Handler() http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		c := cluster.GetCluster(req.Context())
		req.URL.Path = strings.TrimPrefix(req.URL.Path, "/k8s/clusters/"+c.Name)
		e.master.GenericAPIServer.Handler.ServeHTTP(rw, req)
	})
}

func (e *Server) Cluster() *v3.Cluster {
	return e.cluster
}

func New(config *types.GlobalConfig, cluster *v3.Cluster, lookup cluster.Lookup) (*Server, error) {
	storageFactory, err := store.StorageFactory(
		fmt.Sprintf("/k8s/cluster/%s", cluster.Name),
		config)
	if err != nil {
		return nil, err
	}

	clientsetset, err := clients.New(cluster)
	if err != nil {
		return nil, err
	}

	genericAPIServerConfig, err := genericConfig(config, cluster, lookup, storageFactory, clientsetset)
	if err != nil {
		return nil, err
	}

	serviceIPRange, apiServerServiceIP, err := serviceNet(config, cluster)
	if err != nil {
		return nil, errors.Wrap(err, "Invalid service net cidr")
	}

	dialer := proxy.NewDialer(cluster, os.Getenv("CATTLE_ACCESS_KEY"), os.Getenv("CATTLE_SECRET_KEY"))

	masterConfig := &master.Config{
		GenericConfig: genericAPIServerConfig,

		ExtraConfig: master.ExtraConfig{
			APIResourceConfigSource: storageFactory.APIResourceConfigSource,
			StorageFactory:          storageFactory,
			EnableCoreControllers:   true,
			EventTTL:                1 * time.Hour,
			KubeletClientConfig: kubeletclient.KubeletClientConfig{
				Dial:         dialer,
				Port:         ports.KubeletPort,
				ReadOnlyPort: ports.KubeletReadOnlyPort,
				PreferredAddressTypes: []string{
					// --override-hostname
					string(api.NodeHostName),

					// internal, preferring DNS if reported
					string(api.NodeInternalDNS),
					string(api.NodeInternalIP),

					// external, preferring DNS if reported
					string(api.NodeExternalDNS),
					string(api.NodeExternalIP),
				},
				EnableHttps: true,
				HTTPTimeout: time.Duration(5) * time.Second,
			},
			EnableUISupport:   true,
			EnableLogsSupport: true,

			ServiceIPRange:       serviceIPRange,
			APIServerServiceIP:   apiServerServiceIP,
			APIServerServicePort: 443,

			ProxyTransport: &http.Transport{
				Dial: dialer,
			},

			ServiceNodePortRange: utilnet.PortRange{Base: 30000, Size: 2768},

			MasterCount: 1,
		},
	}

	kubeAPIServer, err := masterConfig.Complete(nil).New(genericapiserver.EmptyDelegate)
	kubeAPIServer.GenericAPIServer.AddPostStartHook("start-kube-apiserver-informers", func(context genericapiserver.PostStartHookContext) error {
		clientsetset.Start(context.StopCh)
		return nil
	})
	kubeAPIServer.GenericAPIServer.PrepareRun()

	ctx, cancel := context.WithCancel(context.Background())

	kubeAPIServer.GenericAPIServer.RunPostStartHooks(ctx.Done())
	//go controllermanager.Start(clientsetset, ctx.Done())

	return &Server{
		master:  kubeAPIServer,
		cluster: cluster,
		cancel:  cancel,
	}, nil
}

func serviceNet(config *types.GlobalConfig, cluster *v3.Cluster) (net.IPNet, net.IP, error) {
	cidr := types.FirstNotEmpty(cluster.Spec.EmbeddedConfig.ServiceNetCIDR, config.ServiceNetCidr)
	_, cidrNet, err := net.ParseCIDR(cidr)
	if err != nil {
		return net.IPNet{}, nil, err
	}

	return master.DefaultServiceIPRange(*cidrNet)
}

func genericConfig(config *types.GlobalConfig, cluster *v3.Cluster, lookup *cluster.Lookup,
	storageFactory storage.StorageFactory, clientsetset *clients.ClientSetSet) (*genericapiserver.Config, error) {
	authz := auth.NewAuthorizer(nil)

	admissions, err := admission.New(config, cluster, authz, clientsetset)
	if err != nil {
		return nil, err
	}

	apiVersion := version.Get()

	genericAPIServerConfig := genericapiserver.NewConfig(api.Codecs)
	genericAPIServerConfig.OpenAPIConfig = genericapiserver.DefaultOpenAPIConfig(openapi.GetOpenAPIDefinitions, api.Scheme)
	genericAPIServerConfig.OpenAPIConfig.Info.Title = "Rancher Kubernetes"
	genericAPIServerConfig.OpenAPIConfig.SecurityDefinitions = &spec.SecurityDefinitions{
		"HTTPBasic": &spec.SecurityScheme{
			SecuritySchemeProps: spec.SecuritySchemeProps{
				Type:        "basic",
				Description: "HTTP Basic authentication",
			},
		},
	}
	genericAPIServerConfig.SwaggerConfig = genericapiserver.DefaultSwaggerConfig()
	genericAPIServerConfig.LongRunningFunc = filters.BasicLongRunningRequestCheck(
		sets.NewString("watch", "proxy"),
		sets.NewString("attach", "exec", "proxy", "log", "portforward"),
	)
	genericAPIServerConfig.LoopbackClientConfig = &clientsetset.LoopbackClientConfig
	genericAPIServerConfig.AdmissionControl = admissions
	genericAPIServerConfig.Authorizer = authz
	genericAPIServerConfig.RESTOptionsGetter = &store.RESTOptionsFactory{
		StorageFactory: storageFactory,
	}
	genericAPIServerConfig.Authenticator = auth.NewAuthentication()
	genericAPIServerConfig.Authorizer = authz
	genericAPIServerConfig.PublicAddress = net.ParseIP("169.254.169.250")
	genericAPIServerConfig.ReadWritePort = 9348
	genericAPIServerConfig.EnableDiscovery = true
	genericAPIServerConfig.Version = &apiVersion

	return genericAPIServerConfig, nil
}

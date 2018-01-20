package admission

import (
	"github.com/rancher/netes/clients"
	"github.com/rancher/netes/types"
	"github.com/rancher/types/apis/management.cattle.io/v3"
	"k8s.io/apiserver/pkg/admission"
	"k8s.io/apiserver/pkg/admission/initializer"
	"k8s.io/apiserver/pkg/authorization/authorizer"
	"k8s.io/apiserver/pkg/server"
	"k8s.io/kubernetes/pkg/api"
	kubeapiserveradmission "k8s.io/kubernetes/pkg/kubeapiserver/admission"
	quotainstall "k8s.io/kubernetes/pkg/quota/install"
	"k8s.io/kubernetes/plugin/pkg/admission/admit"
	"k8s.io/kubernetes/plugin/pkg/admission/alwayspullimages"
	"k8s.io/kubernetes/plugin/pkg/admission/antiaffinity"
	"k8s.io/kubernetes/plugin/pkg/admission/defaulttolerationseconds"
	"k8s.io/kubernetes/plugin/pkg/admission/deny"
	"k8s.io/kubernetes/plugin/pkg/admission/exec"
	"k8s.io/kubernetes/plugin/pkg/admission/gc"
	"k8s.io/kubernetes/plugin/pkg/admission/imagepolicy"
	"k8s.io/kubernetes/plugin/pkg/admission/initialization"
	"k8s.io/kubernetes/plugin/pkg/admission/initialresources"
	"k8s.io/kubernetes/plugin/pkg/admission/limitranger"
	"k8s.io/kubernetes/plugin/pkg/admission/namespace/autoprovision"
	"k8s.io/kubernetes/plugin/pkg/admission/namespace/exists"
	"k8s.io/kubernetes/plugin/pkg/admission/noderestriction"
	"k8s.io/kubernetes/plugin/pkg/admission/persistentvolume/label"
	"k8s.io/kubernetes/plugin/pkg/admission/podnodeselector"
	"k8s.io/kubernetes/plugin/pkg/admission/podpreset"
	"k8s.io/kubernetes/plugin/pkg/admission/podtolerationrestriction"
	"k8s.io/kubernetes/plugin/pkg/admission/resourcequota"
	"k8s.io/kubernetes/plugin/pkg/admission/security/podsecuritypolicy"
	"k8s.io/kubernetes/plugin/pkg/admission/securitycontext/scdeny"
	"k8s.io/kubernetes/plugin/pkg/admission/serviceaccount"
	"k8s.io/kubernetes/plugin/pkg/admission/storageclass/setdefault"
	"k8s.io/kubernetes/plugin/pkg/admission/webhook"
)

func New(config *types.GlobalConfig, cluster *v3.Cluster, authz authorizer.Authorizer, clients *clients.ClientSetSet) (admission.Interface, error) {
	pluginInitializer := kubeapiserveradmission.NewPluginInitializer(clients.InternalClient,
		clients.ExternalClient,
		clients.InternalSharedInformers,
		authz,
		nil,
		api.Registry.RESTMapper(),
		quotainstall.NewRegistry(nil, nil))

	names := types.FirstNotLenZero(cluster.Spec.EmbeddedConfig.AdmissionControllers, config.AdmissionControllers)
	pluginsConfigProvider, err := admission.ReadAdmissionConfiguration(names, "")
	if err != nil {
		return nil, err
	}

	genericInitializer, err := initializer.New(clients.Client, clients.SharedInformers, authz)
	if err != nil {
		return nil, err
	}

	return admissionPlugins().NewFromPlugins(names,
		pluginsConfigProvider,
		admission.PluginInitializers{genericInitializer, pluginInitializer})
}

func admissionPlugins() *admission.Plugins {
	plugins := &admission.Plugins{}

	server.RegisterAllAdmissionPlugins(plugins)
	admit.Register(plugins)
	alwayspullimages.Register(plugins)
	antiaffinity.Register(plugins)
	defaulttolerationseconds.Register(plugins)
	deny.Register(plugins)
	exec.Register(plugins)
	gc.Register(plugins)
	imagepolicy.Register(plugins)
	initialization.Register(plugins)
	initialresources.Register(plugins)
	limitranger.Register(plugins)
	autoprovision.Register(plugins)
	exists.Register(plugins)
	noderestriction.Register(plugins)
	label.Register(plugins)
	podnodeselector.Register(plugins)
	podpreset.Register(plugins)
	podtolerationrestriction.Register(plugins)
	resourcequota.Register(plugins)
	podsecuritypolicy.Register(plugins)
	scdeny.Register(plugins)
	serviceaccount.Register(plugins)
	setdefault.Register(plugins)
	webhook.Register(plugins)

	return plugins
}

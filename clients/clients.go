package clients

import (
	"fmt"
	"time"

	"github.com/rancher/go-rancher/v3"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/kubernetes/pkg/client/clientset_generated/clientset"
	"k8s.io/kubernetes/pkg/client/clientset_generated/internalclientset"
	"k8s.io/kubernetes/pkg/client/informers/informers_generated/externalversions"
	"k8s.io/kubernetes/pkg/client/informers/informers_generated/internalversion"
	"k8s.io/kubernetes/pkg/controller"
)

type ClientSetSet struct {
	LoopbackClientConfig rest.Config
	Client               kubernetes.Interface
	SharedInformers      informers.SharedInformerFactory

	ExternalClient          clientset.Interface
	ExternalSharedInformers externalversions.SharedInformerFactory

	InternalClient          internalclientset.Interface
	InternalSharedInformers internalversion.SharedInformerFactory

	ControllerClientBuilder controller.ControllerClientBuilder
}

func (c *ClientSetSet) Start(stopCh <-chan struct{}) {
	c.SharedInformers.Start(stopCh)
	c.ExternalSharedInformers.Start(stopCh)
	c.InternalSharedInformers.Start(stopCh)
}

func New(cluster *client.Cluster) (*ClientSetSet, error) {
	var err error

	c := &ClientSetSet{
		LoopbackClientConfig: rest.Config{
			Host: "http://localhost:8089/k8s/clusters/" + cluster.Id + "/",
			ContentConfig: rest.ContentConfig{
				ContentType: "application/vnd.kubernetes.protobuf",
			},
		},
	}

	c.Client, err = kubernetes.NewForConfig(&c.LoopbackClientConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create clientset: %v", err)
	}
	c.InternalClient, err = internalclientset.NewForConfig(&c.LoopbackClientConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create internal clientset: %v", err)
	}
	c.ExternalClient, err = clientset.NewForConfig(&c.LoopbackClientConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create external clientset: %v", err)
	}

	c.SharedInformers = informers.NewSharedInformerFactory(c.Client, 10*time.Minute)
	c.InternalSharedInformers = internalversion.NewSharedInformerFactory(c.InternalClient, 10*time.Minute)
	c.ExternalSharedInformers = externalversions.NewSharedInformerFactory(c.ExternalClient, 10*time.Minute)

	c.ControllerClientBuilder = controller.SimpleControllerClientBuilder{
		ClientConfig: &c.LoopbackClientConfig,
	}

	return c, err
}

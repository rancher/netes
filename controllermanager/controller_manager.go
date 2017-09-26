package controllermanager

import (
	"time"

	"github.com/golang/glog"
	"github.com/rancher/netes/clients"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/kubernetes/cmd/kube-controller-manager/app"
	"k8s.io/kubernetes/cmd/kube-controller-manager/app/options"
)

func Start(clientsetset *clients.ClientSetSet, stop <-chan struct{}) error {
	// TODO: don't like using cmd/kube-controller-manager/app but the package does too much
	s := options.NewCMServer()

	availableResources, err := app.GetAvailableResources(clientsetset.ControllerClientBuilder)
	if err != nil {
		return err
	}

	// TODO: Init cloud provider?
	//cloud, err := cloudprovider.InitCloudProvider(s.CloudProvider, s.CloudConfigFile)
	//if err != nil {
	//	return ControllerContext{}, fmt.Errorf("cloud provider could not be initialized: %v", err)
	//}
	//if cloud != nil {
	//	// Initialize the cloud provider with a reference to the clientBuilder
	//	cloud.Initialize(rootClientBuilder)
	//}

	ctx := app.ControllerContext{
		ClientBuilder:      clientsetset.ControllerClientBuilder,
		InformerFactory:    clientsetset.ExternalSharedInformers,
		Options:            *s,
		AvailableResources: availableResources,
		Cloud:              nil,
		Stop:               stop,
	}

	return startControllers(ctx)
}

func startControllers(ctx app.ControllerContext) error {
	for controllerName, initFn := range app.NewControllerInitializers() {
		if !ctx.IsControllerEnabled(controllerName) {
			continue
		}

		time.Sleep(wait.Jitter(ctx.Options.ControllerStartInterval.Duration, app.ControllerStartJitter))

		glog.V(1).Infof("Starting %q", controllerName)
		started, err := initFn(ctx)
		if err != nil {
			glog.Errorf("Error starting %q", controllerName)
			return err
		}
		if !started {
			glog.Warningf("Skipping %q", controllerName)
			continue
		}
		glog.Infof("Started %q", controllerName)
	}

	return nil
}

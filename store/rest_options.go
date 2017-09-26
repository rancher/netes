package store

import (
	"fmt"

	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apiserver/pkg/registry/generic"
	"k8s.io/apiserver/pkg/server/storage"
)

type RESTOptionsFactory struct {
	StorageFactory storage.StorageFactory
}

func (f *RESTOptionsFactory) GetRESTOptions(resource schema.GroupResource) (generic.RESTOptions, error) {
	storageConfig, err := f.StorageFactory.NewConfig(resource)
	if err != nil {
		return generic.RESTOptions{}, fmt.Errorf("unable to find storage destination for %v, due to %v", resource, err.Error())
	}

	ret := generic.RESTOptions{
		StorageConfig: storageConfig,
		//Decorator:     registry.StorageWithCacher(100),
		Decorator:               generic.UndecoratedStorage,
		DeleteCollectionWorkers: 1,
		EnableGarbageCollection: true,
		ResourcePrefix:          f.StorageFactory.ResourcePrefix(resource),
	}

	return ret, nil
}

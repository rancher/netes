package store

import (
	"github.com/rancher/k8s-sql"
	// Include MySQL dialect for k8s-sql
	_ "github.com/rancher/k8s-sql/dialect/mysql"
	"github.com/rancher/netes/types"
	serverstorage "k8s.io/apiserver/pkg/server/storage"
	"k8s.io/apiserver/pkg/storage/storagebackend"
	"k8s.io/apiserver/pkg/storage/storagebackend/factory"
	"k8s.io/apiserver/pkg/util/flag"
	"k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/kubeapiserver"
	"k8s.io/kubernetes/pkg/master"
)

const StorageTypeRDBMS = "mysql"

func init() {
	factory.Register(StorageTypeRDBMS, rdbms.NewRDBMSStorage)
}

func StorageFactory(pathPrefix string, config *types.GlobalConfig) (*serverstorage.DefaultStorageFactory, error) {
	storageConfig := storagebackend.NewDefaultConfig(pathPrefix, api.Scheme, nil)
	storageConfig.Type = StorageTypeRDBMS
	storageConfig.ServerList = []string{
		config.Dialect,
		config.DSN,
	}

	return kubeapiserver.NewStorageFactory(
		*storageConfig,
		"application/vnd.kubernetes.protobuf",
		api.Codecs,
		serverstorage.NewDefaultResourceEncodingConfig(api.Registry),
		nil,
		nil,
		master.DefaultAPIResourceConfigSource(),
		flag.ConfigurationMap{
			"api/all": "true",
		})
}

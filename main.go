package main

import (
	"fmt"
	"os"

	"github.com/rancher/netes/master"
	"github.com/rancher/netes/store"
	"github.com/rancher/netes/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apiserver/pkg/util/logs"
)

func main() {
	utilruntime.ReallyCrash = false
	logs.InitLogs()

	dsn := os.Getenv("NETES_DB_DSN")
	if dsn == "" {
		user := getenv("NETES_MYSQL_USER", "cattle")
		password := getenv("NETES_MYSQL_PASSWORD", "cattle")
		address := getenv("NETES_MYSQL_ADDRESS", "localhost:3306")
		dbName := getenv("NETES_MYSQL_DBNAME", "cattle")
		params := getenv("NETES_MYSQL_PARAMS", "")

		dsn = store.FormatDSN(
			user,
			password,
			address,
			dbName,
			params,
		)
	}

	err := master.New(&types.GlobalConfig{
		Dialect:    "mysql",
		DSN:        dsn,
		CattleURL:  "http://localhost:8081/v3/",
		ListenAddr: ":8089",
		AdmissionControllers: []string{
			"NamespaceLifecycle",
			"LimitRanger",
			"ServiceAccount",
			"PersistentVolumeLabel",
			"DefaultStorageClass",
			"ResourceQuota",
			"DefaultTolerationSeconds",
		},
		ServiceNetCidr: "10.43.0.0/24",
	}).Run()

	fmt.Fprintf(os.Stdout, "Failed to run netes: %v", err)
	os.Exit(1)
}

func getenv(key, def string) string {
	val := os.Getenv(key)
	if val == "" {
		return def
	}
	return val
}

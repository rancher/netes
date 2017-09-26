package server

import (
	"net/http"

	"github.com/rancher/go-rancher/v3"
)

type Server interface {
	Close()
	Handler() http.Handler
	Cluster() *client.Cluster
}

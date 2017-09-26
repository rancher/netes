package authentication

import (
	"fmt"
	"net/http"

	"github.com/rancher/netes/cluster"
	"k8s.io/apiserver/pkg/authentication/authenticator"
	"k8s.io/apiserver/pkg/authentication/group"
	"k8s.io/apiserver/pkg/authentication/user"
)

type Authenticator struct {
	clusterLookup *cluster.Lookup
}

func New(clusterLookup *cluster.Lookup) authenticator.Request {
	return group.NewAuthenticatedGroupAdder(&Authenticator{
		clusterLookup: clusterLookup,
	})
}

func (a *Authenticator) AuthenticateRequest(req *http.Request) (user.Info, bool, error) {
	c := cluster.GetCluster(req.Context())
	if c == nil {
		return nil, false, nil
	}

	attrs := map[string][]string{}
	for k, v := range c.Identity.Attributes {
		attrs[k] = []string{fmt.Sprint(v)}
	}

	return &user.DefaultInfo{
		Name:   c.Identity.Username,
		UID:    c.Identity.UserId,
		Groups: []string{"system:masters"},
		Extra:  attrs,
	}, true, nil
}

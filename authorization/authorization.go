package authorization

import (
	authz "k8s.io/apiserver/pkg/authorization/authorizer"
)

type authorizer struct {
}

func New() (authz.Authorizer, error) {
	return &authorizer{}, nil
}

func (a *authorizer) Authorize(attr authz.Attributes) (authorized bool, reason string, err error) {
	return true, "", nil
}

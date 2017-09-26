package types

import "github.com/rancher/netes/cluster"

type GlobalConfig struct {
	Dialect    string
	DSN        string
	CattleURL  string
	ListenAddr string

	AdmissionControllers []string
	ServiceNetCidr       string

	Lookup *cluster.Lookup
}

func FirstNotEmpty(left, right string) string {
	if left != "" {
		return left
	}
	return right
}

func FirstNotLenZero(left, right []string) []string {
	if len(left) > 0 {
		return left
	}
	return right
}

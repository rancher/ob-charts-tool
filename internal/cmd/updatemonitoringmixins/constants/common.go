package constants

import "github.com/rancher/ob-charts-tool/internal/cmd/updatemonitoringmixins/git"

var Repos = map[string]git.RepoConfigStatus{
	"etcd": {
		Name:    "etcd",
		RepoURL: "https://github.com/etcd-io/etcd.git",
		Branch:  "main",
		// HeadSha: "7351ab86c054aad7d31d6639b2e841f2c37cd296", // If HeadSha is set we will use that specific SHA over branch
	},
	"kube-prometheus": {
		Name:    "kube-prometheus",
		RepoURL: "https://github.com/prometheus-operator/kube-prometheus.git",
		Branch:  "main",
		// HeadSha: "685008710cbb881cd8fce9db1e2f890c9e249903", // If HeadSha is set we will use that specific SHA over branch
	},
	"kubernetes-mixin": {
		Name:    "kubernetes-mixin",
		RepoURL: "https://github.com/kubernetes-monitoring/kubernetes-mixin.git",
		Branch:  "master",
		// HeadSha: "834daaa30905d5832c68b7ef8ab41fbedcd9dd4b", // If HeadSha is set we will use that specific SHA over branch
	},
	// TODO: in the future we'll maybe add a Rancher source
}

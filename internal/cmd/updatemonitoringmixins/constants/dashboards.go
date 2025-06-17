package constants

import (
	"bytes"
	"fmt"
	"github.com/rancher/ob-charts-tool/internal/cmd/updatemonitoringmixins/git"
	"github.com/rancher/ob-charts-tool/internal/cmd/updatemonitoringmixins/pythonish"
	"github.com/rancher/ob-charts-tool/internal/cmd/updatemonitoringmixins/types"
)

var Repos = map[string]git.RepoConfigStatus{
	"kube-prometheus": {
		Name:    "kube-prometheus",
		RepoURL: "https://github.com/prometheus-operator/kube-prometheus.git",
		HeadSha: "685008710cbb881cd8fce9db1e2f890c9e249903",
	},
	"kubernetes-mixin": {
		Name:    "kubernetes-mixin",
		RepoURL: "https://github.com/kubernetes-monitoring/kubernetes-mixin.git",
		HeadSha: "834daaa30905d5832c68b7ef8ab41fbedcd9dd4b",
	},
	"etcd": {
		Name:    "etcd",
		RepoURL: "https://github.com/etcd-io/etcd.git",
		HeadSha: "7351ab86c054aad7d31d6639b2e841f2c37cd296",
	},
	// TODO: in the future we'll maybe add a Rancher source
}

func SourceCharts(chartRefs map[string]string) types.DashboardConfig {
	return types.DashboardConfig{
		types.DashboardFileSource{
			Source: "/files/dashboards/k8s-coredns.json",
			DashboardSourceBase: types.DashboardSourceBase{
				Destination:     "/templates/grafana/dashboards-1.14",
				Type:            types.DashboardJson,
				MinKubernetes:   "1.14.0-0",
				MulticlusterKey: ".Values.grafana.sidecar.dashboards.multicluster.global.enabled",
			},
		},
		types.DashboardURLSource{
			Source: fmt.Sprintf("https://raw.githubusercontent.com/prometheus-operator/kube-prometheus/%s/manifests/grafana-dashboardDefinitions.yaml", chartRefs["kube-prometheus"]),
			DashboardSourceBase: types.DashboardSourceBase{
				Destination:     "/templates/grafana/dashboards-1.14",
				Type:            types.DashboardYaml,
				MinKubernetes:   "1.14.0-0",
				MulticlusterKey: ".Values.grafana.sidecar.dashboards.multicluster.global.enabled",
			},
		},
		types.DashboardGitSource{
			Repository: Repos["kubernetes-mixin"],
			Branch:     chartRefs["kubernetes-mixin"],
			// TODO: figure out how to replace these mixins from python...lol
			// Or maybe these are rendered by something else?
			Content: "(import 'dashboards/windows.libsonnet') + (import 'config.libsonnet') + { _config+:: { windowsExporterSelector: 'job=\"windows-exporter\"', }}",
			Cwd:     ".",
			Source:  "_mixin.jsonnet",
			DashboardSourceBase: types.DashboardSourceBase{
				Destination:     "/templates/grafana/dashboards-1.14",
				MinKubernetes:   "1.14.0-0",
				Type:            types.DashboardJsonnetMixin,
				MulticlusterKey: ".Values.grafana.sidecar.dashboards.multicluster.global.enabled",
			},
			MixinVars: map[string]interface{}{},
		},
		types.DashboardGitSource{
			Repository: Repos["etcd"],
			Branch:     chartRefs["etcd"],
			Source:     "mixin.libsonnet",
			Cwd:        "contrib/mixin",
			DashboardSourceBase: types.DashboardSourceBase{
				Destination:     "/templates/grafana/dashboards-1.14",
				MinKubernetes:   "1.14.0-0",
				Type:            types.DashboardJsonnetMixin,
				MulticlusterKey: "(or .Values.grafana.sidecar.dashboards.multicluster.global.enabled .Values.grafana.sidecar.dashboards.multicluster.etcd.enabled)",
			},
			MixinVars: map[string]interface{}{
				"_config+": map[string]interface{}{},
			},
		},
	}
}

var ConditionMap = map[string]string{
	"alertmanager-overview":           " (or .Values.alertmanager.enabled .Values.alertmanager.forceDeployDashboards)",
	"grafana-coredns-k8s":             " .Values.coreDns.enabled",
	"etcd":                            " .Values.kubeEtcd.enabled",
	"apiserver":                       " .Values.kubeApiServer.enabled",
	"controller-manager":              " .Values.kubeControllerManager.enabled",
	"kubelet":                         " .Values.kubelet.enabled",
	"proxy":                           " .Values.kubeProxy.enabled",
	"scheduler":                       " .Values.kubeScheduler.enabled",
	"node-rsrc-use":                   " (or .Values.nodeExporter.enabled .Values.nodeExporter.forceDeployDashboards)",
	"node-cluster-rsrc-use":           " (or .Values.nodeExporter.enabled .Values.nodeExporter.forceDeployDashboards)",
	"nodes":                           " (and (or .Values.nodeExporter.enabled .Values.nodeExporter.forceDeployDashboards) .Values.nodeExporter.operatingSystems.linux.enabled)",
	"nodes-aix":                       " (and (or .Values.nodeExporter.enabled .Values.nodeExporter.forceDeployDashboards) .Values.nodeExporter.operatingSystems.aix.enabled)",
	"nodes-darwin":                    " (and (or .Values.nodeExporter.enabled .Values.nodeExporter.forceDeployDashboards) .Values.nodeExporter.operatingSystems.darwin.enabled)",
	"prometheus-remote-write":         " .Values.prometheus.prometheusSpec.remoteWriteDashboards",
	"k8s-coredns":                     " .Values.coreDns.enabled",
	"k8s-windows-cluster-rsrc-use":    " .Values.windowsMonitoring.enabled",
	"k8s-windows-node-rsrc-use":       " .Values.windowsMonitoring.enabled",
	"k8s-resources-windows-cluster":   " .Values.windowsMonitoring.enabled",
	"k8s-resources-windows-namespace": " .Values.windowsMonitoring.enabled",
	"k8s-resources-windows-pod":       " .Values.windowsMonitoring.enabled",
}

const StandardHeader = `{{- /*
Generated from '%(.Name)s' from ..%(.URL)s by 'ob-charts-tool'
Do not change in-place! In order to change this file first read following link:
https://github.com/prometheus-community/helm-charts/tree/main/charts/kube-prometheus-stack/hack
*/ -}}
{{- $kubeTargetVersion := default .Capabilities.KubeVersion.GitVersion .Values.kubeTargetVersionOverride }}
{{- if and (or .Values.grafana.enabled .Values.grafana.forceDeployDashboards) (semverCompare ">=%(.MinKubeVersion)s" $kubeTargetVersion) (semverCompare "<%(.MaxKubeVersion)s" $kubeTargetVersion) .Values.grafana.defaultDashboardsEnabled%(.Condition)s }}
apiVersion: v1
kind: ConfigMap
metadata:
  namespace: {{ template "kube-prometheus-stack-grafana.namespace" . }}
  name: {{ printf "%s-%s" (include "kube-prometheus-stack.fullname" $) "%(.Name)s" | trunc 63 | trimSuffix "-" }}
  annotations:
{{ toYaml .Values.grafana.sidecar.dashboards.annotations | indent 4 }}
  labels:
    {{- if $.Values.grafana.sidecar.dashboards.label }}
    {{ $.Values.grafana.sidecar.dashboards.label }}: {{ ternary $.Values.grafana.sidecar.dashboards.labelValue "1" (not (empty $.Values.grafana.sidecar.dashboards.labelValue)) | quote }}
    {{- end }}
    app: {{ template "kube-prometheus-stack.name" $ }}-grafana
{{ include "kube-prometheus-stack.labels" $ | indent 4 }}
data:
`

type HeaderData struct {
	Name           string
	URL            string
	Condition      any
	MinKubeVersion string
	MaxKubeVersion string
}

func NewHeader(headerData HeaderData) (string, error) {
	templateRenderer := pythonish.NewRenderer()
	tmpl, err := templateRenderer.Parse(StandardHeader)
	if err != nil {
		return "ERROR", err
	}

	var buffer bytes.Buffer
	err = tmpl.Execute(&buffer, headerData)
	return buffer.String(), err
}

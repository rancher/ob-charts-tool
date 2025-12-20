package testNewMonitoringVersion

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"sort"
	"time"

	"github.com/Masterminds/semver/v3"
	"gopkg.in/yaml.v2"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/tools/clientcmd"
)

// AppState defines the desired state of an application.
type AppState string

const (
	// AppStateDeployed is the state for a successfully deployed application.
	AppStateDeployed AppState = "deployed"
	// AppStateDeleted is the state for a successfully deleted application.
	AppStateDeleted AppState = "deleted"
)

// HelmIndex represents the structure of a Helm repository index file.
type HelmIndex struct {
	Entries map[string][]HelmChart `yaml:"entries"`
}

// HelmChart represents a single chart entry in the index file.
type HelmChart struct {
	Version string `yaml:"version"`
}

// InstallPayload represents the JSON body for the install request.
type InstallPayload struct {
	Charts                   []Chart     `json:"charts"`
	NoHooks                  bool        `json:"noHooks"`
	Timeout                  string      `json:"timeout"`
	Wait                     bool        `json:"wait"`
	Namespace                string      `json:"namespace"`
	ProjectID                interface{} `json:"projectId"`
	DisableOpenAPIValidation bool        `json:"disableOpenAPIValidation"`
	SkipCRDs                 bool        `json:"skipCRDs"`
}

// Chart represents a single chart to be installed.
type Chart struct {
	ChartName   string                 `json:"chartName"`
	Version     string                 `json:"version"`
	ReleaseName string                 `json:"releaseName"`
	ProjectID   interface{}            `json:"projectId,omitempty"`
	Values      map[string]interface{} `json:"values"`
	Annotations map[string]string      `json:"annotations,omitempty"`
}

// GetPreviousVersion fetches the Helm index and finds the version prior to the one specified.
func GetPreviousVersion(currentVersion, rancherURL, sessionToken string) (string, error) {
	indexURL := fmt.Sprintf("%s/v1/catalog.cattle.io.clusterrepos/rancher-charts?link=index", rancherURL)

	req, err := http.NewRequest("GET", indexURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request for helm index: %w", err)
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("Cookie", fmt.Sprintf("R_SESS=%s", sessionToken))
	req.Header.Set("User-Agent", "ob-charts-tool")

	tr := &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}
	client := &http.Client{Transport: tr}

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to fetch helm index: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		body, _ := ioutil.ReadAll(resp.Body)
		return "", fmt.Errorf("failed to fetch helm index, status code: %d, body: %s", resp.StatusCode, string(body))
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read helm index body: %w", err)
	}

	var index HelmIndex
	if err := yaml.Unmarshal(body, &index); err != nil {
		return "", fmt.Errorf("failed to unmarshal helm index: %w", err)
	}

	chartEntries, ok := index.Entries["rancher-monitoring"]
	if !ok {
		return "", fmt.Errorf("chart 'rancher-monitoring' not found in helm index")
	}

	var versions []*semver.Version
	for _, entry := range chartEntries {
		v, err := semver.NewVersion(entry.Version)
		if err != nil {
			continue
		}
		versions = append(versions, v)
	}

	sort.Sort(sort.Reverse(semver.Collection(versions)))

	targetVersion, err := semver.NewVersion(currentVersion)
	if err != nil {
		return "", fmt.Errorf("invalid current version format: %w", err)
	}

	for _, v := range versions {
		if v.LessThan(targetVersion) {
			return v.Original(), nil
		}
	}

	return "", nil // No previous version found
}

// InstallCurrentVersion installs the rancher-monitoring chart for a given version.
func InstallCurrentVersion(chartVersion, rancherURL, sessionToken string) error {
	url := fmt.Sprintf("%s/v1/catalog.cattle.io.clusterrepos/rancher-charts?action=install", rancherURL)

	payload := InstallPayload{
		Charts: []Chart{
			{
				ChartName:   "rancher-monitoring-crd",
				Version:     chartVersion,
				ReleaseName: "rancher-monitoring-crd",
				Values: map[string]interface{}{
					"global": map[string]interface{}{"cattle": map[string]interface{}{"systemDefaultRegistry": "", "clusterId": "local", "clusterName": "local", "url": rancherURL}},
				},
			},
			{
				ChartName:   "rancher-monitoring",
				Version:     chartVersion,
				ReleaseName: "rancher-monitoring",
				Annotations: map[string]string{"catalog.cattle.io/ui-source-repo-type": "cluster", "catalog.cattle.io/ui-source-repo": "rancher-charts"},
				Values: map[string]interface{}{
					"global":               map[string]interface{}{"cattle": map[string]interface{}{"systemDefaultRegistry": "", "clusterId": "local", "clusterName": "local", "url": rancherURL}},
					"prometheus":           map[string]interface{}{"prometheusSpec": map[string]interface{}{"resources": map[string]interface{}{"requests": map[string]interface{}{"memory": "1750Mi"}}, "retentionSize": "50GiB"}},
					"k3sServer":            map[string]interface{}{"enabled": true},
					"k3sControllerManager": map[string]interface{}{"enabled": true},
					"k3sScheduler":         map[string]interface{}{"enabled": true},
					"k3sProxy":             map[string]interface{}{"enabled": true},
				},
			},
		},
		NoHooks:                  false,
		Timeout:                  "600s",
		Wait:                     true,
		Namespace:                "cattle-monitoring-system",
		DisableOpenAPIValidation: false,
		SkipCRDs:                 false,
	}

	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonPayload))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Cookie", fmt.Sprintf("R_SESS=%s", sessionToken))
	req.Header.Set("User-Agent", "ob-charts-tool")

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		body, _ := ioutil.ReadAll(resp.Body)
		return fmt.Errorf("failed to install chart, status code: %d, body: %s", resp.StatusCode, string(body))
	}

	fmt.Printf("Successfully sent install request for chart version %s. Waiting for deployment...\n", chartVersion)

	time.Sleep(30 * time.Second)
	err = waitForAppState("rancher-monitoring", "cattle-monitoring-system", AppStateDeployed)
	if err != nil {
		return fmt.Errorf("failed to wait for app deployment: %w", err)
	}

	fmt.Printf("Successfully deployed chart version %s\n", chartVersion)
	return nil
}

// UninstallChart sends a request to uninstall a chart and waits for its App resource to be deleted.
func UninstallChart(chartName, namespace, rancherURL, sessionToken string) error {
	uninstallURL := fmt.Sprintf("%s/v1/catalog.cattle.io.apps/%s/%s?action=uninstall", rancherURL, namespace, chartName)

	req, err := http.NewRequest("POST", uninstallURL, bytes.NewBufferString("{}"))
	if err != nil {
		return fmt.Errorf("failed to create uninstall request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Cookie", fmt.Sprintf("R_SESS=%s", sessionToken))
	req.Header.Set("User-Agent", "ob-charts-tool")

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to execute uninstall request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		body, _ := ioutil.ReadAll(resp.Body)
		return fmt.Errorf("failed to uninstall chart, status code: %d, body: %s", resp.StatusCode, string(body))
	}

	fmt.Printf("Successfully sent uninstall request for chart %s/%s. Waiting for deletion...\n", namespace, chartName)

	if err := waitForAppState(chartName, namespace, AppStateDeleted); err != nil {
		return err
	}

	fmt.Printf("Successfully uninstalled chart %s/%s.\n", namespace, chartName)
	return nil
}

// waitForAppState polls the status of a catalog.cattle.io/v1 App until it reaches the desired state.
func waitForAppState(name, namespace string, desiredState AppState) error {
	config, err := clientcmd.BuildConfigFromFlags("", clientcmd.RecommendedHomeFile)
	if err != nil {
		return fmt.Errorf("failed to build kubeconfig: %w", err)
	}

	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		return fmt.Errorf("failed to create dynamic client: %w", err)
	}

	appResource := schema.GroupVersionResource{
		Group:    "catalog.cattle.io",
		Version:  "v1",
		Resource: "apps",
	}

	timeout := time.After(5 * time.Minute)
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-timeout:
			return fmt.Errorf("timed out waiting for app %s/%s to reach state %s", namespace, name, desiredState)
		case <-ticker.C:
			app, err := dynamicClient.Resource(appResource).Namespace(namespace).Get(context.TODO(), name, metav1.GetOptions{})
			if err != nil {
				if errors.IsNotFound(err) {
					if desiredState == AppStateDeleted {
						fmt.Printf("... app %s/%s has been deleted.\n", namespace, name)
						return nil // Success for deletion
					}
				}
				fmt.Printf("... failed to get app %s/%s, retrying: %v\n", namespace, name, err)
				continue
			}

			if desiredState == AppStateDeployed {
				summary, found, err := unstructured.NestedMap(app.Object, "status", "summary")
				if err != nil || !found {
					fmt.Printf("... status.summary not found for app %s/%s, retrying.\n", namespace, name)
					continue
				}

				state, found, err := unstructured.NestedString(summary, "state")
				if err != nil || !found {
					fmt.Printf("... status.summary.state not found for app %s/%s, retrying.\n", namespace, name)
					continue
				}

				fmt.Printf("... current state of app %s/%s is '%s'\n", namespace, name, state)
				if state == string(desiredState) {
					return nil
				}
			} else {
				fmt.Printf("... waiting for app %s/%s to be deleted.\n", namespace, name)
			}
		}
	}
}

package testnewmonitoringversion

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"time"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

const (
	dashboardNamespace = "cattle-dashboards"
)

var ignoredConfigMaps = []string{"kube-root-ca.crt", "rancher-fleet-dashboards"}

// PrometheusResponse represents the structure of the Prometheus API response.
type PrometheusResponse struct {
	Status    string         `json:"status"`
	Data      PrometheusData `json:"data,omitempty"`
	Error     string         `json:"error,omitempty"`
	ErrorType string         `json:"errorType,omitempty"`
}

// PrometheusData represents the data field within the Prometheus API response.
type PrometheusData struct {
	ResultType string        `json:"resultType"`
	Result     []interface{} `json:"result"`
}

// QueryResult represents the result of a single Prometheus query execution.
type QueryResult struct {
	Expr         string `json:"expr"`
	Status       string `json:"status"`
	Error        string `json:"error,omitempty"`
	ErrorType    string `json:"errorType,omitempty"`
	DataInResult bool   `json:"dataInResult"`
}

// PanelTestResult represents the test results for a single dashboard panel.
type PanelTestResult struct {
	Panel   string        `json:"panel"`
	Results []QueryResult `json:"results"`
}

// dashboardTemplateVars holds the dynamic values for query interpolation.
type dashboardTemplateVars struct {
	Namespace    string
	Cluster      string
	Instance     string
	Node         string
	Pod          string
	RateInterval string
	Interval     string
	Resolution   string
}

// GetDashboards retrieves all Grafana dashboards stored in ConfigMaps.
func GetDashboards() (map[string]interface{}, error) {
	config, err := clientcmd.BuildConfigFromFlags("", clientcmd.RecommendedHomeFile)
	if err != nil {
		return nil, fmt.Errorf("failed to build kubeconfig: %w", err)
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create clientset: %w", err)
	}

	cmList, err := clientset.CoreV1().ConfigMaps(dashboardNamespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list configmaps in namespace %s: %w", dashboardNamespace, err)
	}

	allDashboards := make(map[string]interface{})

	for _, cm := range cmList.Items {
		if slices.Contains(ignoredConfigMaps, cm.Name) {
			continue
		}

		for name, dashboardJSON := range cm.Data {
			var dashboardData interface{}
			if err := json.Unmarshal([]byte(dashboardJSON), &dashboardData); err != nil {
				fmt.Printf("... skipping dashboard '%s' from ConfigMap '%s' due to JSON parsing error: %v\n", name, cm.Name, err)
				continue
			}
			allDashboards[name] = dashboardData
		}
	}

	fmt.Printf("Successfully loaded %d dashboards from namespace %s.\n", len(allDashboards), dashboardNamespace)
	return allDashboards, nil
}

// getTemplateVars fetches dynamic values from the k8s cluster for query interpolation.
func getTemplateVars(clientset *kubernetes.Clientset, templatingList []interface{}) (*dashboardTemplateVars, error) {
	vars := &dashboardTemplateVars{
		Namespace:    "cattle-monitoring-system",
		Cluster:      "local",
		RateInterval: "2m0s", //default is 4x the prometheus scrap time, and we use 30s
	}

	for _, t := range templatingList {
		template, found := t.(map[string]interface{})
		if !found {
			continue
		}
		name, found := template["name"].(string)
		if !found {
			continue
		}
		options, found := template["options"].([]interface{})
		if !found || len(options) == 0 {
			continue
		}
		if name == "resolution" {
			vars.Resolution = options[0].(map[string]interface{})["value"].(string)
		} else if name == "interval" {
			vars.Interval = options[0].(map[string]interface{})["value"].(string)
		}
	}

	// Get Node name
	nodes, err := clientset.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{})
	if err != nil || len(nodes.Items) == 0 {
		return nil, fmt.Errorf("failed to get cluster node: %w", err)
	}
	// Assuming a single-node cluster
	node := nodes.Items[0]
	vars.Node = node.Name
	for _, addr := range node.Status.Addresses {
		if addr.Type == v1.NodeInternalIP {
			// default port of node-exporter we use is 9796
			vars.Instance = fmt.Sprintf("%s:9796", addr.Address)
			break
		}
	}
	if vars.Instance == "" {
		return nil, fmt.Errorf("could not find internal IP for node %s", vars.Node)
	}

	// Get Grafana pod name
	pods, err := clientset.CoreV1().Pods("cattle-monitoring-system").List(context.TODO(), metav1.ListOptions{
		LabelSelector: "app.kubernetes.io/name=grafana",
	})
	if err != nil || len(pods.Items) == 0 {
		return nil, fmt.Errorf("failed to find grafana pod: %w", err)
	}
	vars.Pod = pods.Items[0].Name

	return vars, nil
}

// TestDashboard executes all queries within a given dashboard against the Prometheus API.
func TestDashboard(dashboard map[string]interface{}, rancherURL, sessionToken string) ([]PanelTestResult, error) {
	config, err := clientcmd.BuildConfigFromFlags("", clientcmd.RecommendedHomeFile)
	if err != nil {
		return nil, fmt.Errorf("failed to build kubeconfig for TestDashboard: %w", err)
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create clientset for TestDashboard: %w", err)
	}
	var templatingList []interface{}
	templating, found := dashboard["templating"].(map[string]interface{})
	if found {
		templatingList, found = templating["list"].([]interface{})
		if !found {
			fmt.Println(found)
		}
	}
	templateVars, err := getTemplateVars(clientset, templatingList)
	if err != nil {
		return nil, fmt.Errorf("failed to get dynamic template variables: %w", err)
	}

	replacer := strings.NewReplacer(
		"$namespace", templateVars.Namespace,
		"$cluster", templateVars.Cluster,
		"$instance", templateVars.Instance,
		"$node", templateVars.Node,
		"$pod", templateVars.Pod,
		"$__rate_interval", templateVars.RateInterval,
		"$resolution", templateVars.Resolution,
		"$interval", templateVars.Interval,
	)

	var allPanelResults []PanelTestResult
	panels, found := dashboard["panels"].([]interface{})
	if !found {
		rows, found := dashboard["rows"].([]interface{})
		if !found {
			fmt.Println("dashboard does not contain a 'panels' field or it's not an array")
			return nil, nil
		}
		panels, found = rows[0].(map[string]interface{})["panels"].([]interface{})
		if !found {
			fmt.Println("dashboard does not contain a 'panels' field or it's not an array")
			return nil, nil
		}
	}

	tr := &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}
	client := &http.Client{Transport: tr}

	for _, panelIface := range panels {
		panel, _ := panelIface.(map[string]interface{})
		panelTitle, _ := panel["title"].(string)
		if panelTitle == "" {
			panelTitle = "Untitled Panel"
		}

		var currentPanelResults []QueryResult
		targets, _ := panel["targets"].([]interface{})

		for _, targetIface := range targets {
			target, _ := targetIface.(map[string]interface{})
			expr, _ := target["expr"].(string)
			if expr == "" {
				continue
			}

			finalExpr := replacer.Replace(expr)

			prometheusQueryURL := fmt.Sprintf("%s/k8s/clusters/local/api/v1/namespaces/cattle-monitoring-system/services/http:rancher-monitoring-prometheus:9090/proxy/api/v1/query", rancherURL)
			currentTime := time.Now().Unix()
			fullQueryURL := fmt.Sprintf("%s?query=%s&time=%d", prometheusQueryURL, url.QueryEscape(finalExpr), currentTime)

			req, _ := http.NewRequest("GET", fullQueryURL, nil)
			req.Header.Set("Accept", "application/json")
			req.Header.Set("Cookie", fmt.Sprintf("R_SESS=%s", sessionToken))
			req.Header.Set("User-Agent", "ob-charts-tool")

			resp, err := client.Do(req)
			if err != nil {
				continue
			}
			defer resp.Body.Close()

			body, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				currentPanelResults = append(currentPanelResults, QueryResult{
					Expr:   expr,
					Status: "error",
					Error:  fmt.Sprintf("failed to read response body: %v", err),
				})
				continue
			}

			var promResp PrometheusResponse
			json.Unmarshal(body, &promResp)

			queryRes := QueryResult{
				Expr:   finalExpr,
				Status: promResp.Status,
			}
			if promResp.Status == "error" {
				queryRes.Error = promResp.Error
				queryRes.ErrorType = promResp.ErrorType
			} else {
				queryRes.DataInResult = len(promResp.Data.Result) > 0
			}
			currentPanelResults = append(currentPanelResults, queryRes)
		}

		if len(currentPanelResults) > 0 {
			allPanelResults = append(allPanelResults, PanelTestResult{
				Panel:   panelTitle,
				Results: currentPanelResults,
			})
		}
	}

	return allPanelResults, nil
}

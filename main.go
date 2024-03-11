package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

type Config struct {
	PrometheusServer string          `yaml:"prometheus_server"`
	TotalVsErrors    []TotalVsErrors `yaml:"total_vs_errors"`
	Ups              []Up            `yaml:"up"`
}

type Up struct {
	Namespace string `yaml:"namespace"`
}

type TotalVsErrors struct {
	Name       string `yaml:"name"`
	TotalQuery string `yaml:"total_query"`
	ErrorQuery string `yaml:"error_query"`
}

const queryURLFormat = "%s/api/v1/query?query=%s"

type QueryResponse struct {
	Status string `json:"status"`
	Data   Data   `json:"data"`
}

type Data struct {
	ResultType string   `json:"resultType"`
	Result     []Metric `json:"result"`
}

type Metric struct {
	Labels map[string]string `json:"metric"`
	Value  []interface{}     `json:"value"`
}

type TotalVsErrorsResult struct {
	Name       string
	TotalRate  float64
	ErrorRate  float64
	ErrorRatio float64
}

type UpResult struct {
	Namespace string
	Value     int
}

func main() {
	var config Config
	contents, err := os.ReadFile("./config.yaml")
	if err != nil {
		fatal(fmt.Sprintf("Could not read ./config.yaml: %s\n", err.Error()))
	}
	yaml.Unmarshal(contents, &config)
	var totalVsErrorsResults []TotalVsErrorsResult
	for _, app := range config.TotalVsErrors {
		totalRate, err := metricQuery(config.PrometheusServer, app.TotalQuery)
		if err != nil {
			fatal(fmt.Sprintf("Could not make query %s: %s", app.TotalQuery, err.Error()))
		}
		errorRate, err := metricQuery(config.PrometheusServer, app.ErrorQuery)
		if err != nil {
			fatal(fmt.Sprintf("Could not make query %s: %s", app.ErrorQuery, err.Error()))
		}
		totalVsErrorsResults = append(totalVsErrorsResults, TotalVsErrorsResult{
			Name:       app.Name,
			TotalRate:  totalRate,
			ErrorRate:  errorRate,
			ErrorRatio: errorRate / totalRate,
		})
	}
	longestAppName := 0
	for _, result := range totalVsErrorsResults {
		if len(result.Name) > longestAppName {
			longestAppName = len(result.Name)
		}
	}
	lines := []string{fmt.Sprintf("%sRequests/s  Errors/s  Error ratio", strings.Repeat(" ", longestAppName+2))}
	for _, row := range totalVsErrorsResults {
		lines = append(lines, fmt.Sprintf("%-*s%f    %f  %f", longestAppName+2, row.Name, row.TotalRate, row.ErrorRate, row.ErrorRatio))
	}
	fmt.Println(strings.Join(lines, "\n"))
	fmt.Println()

	var upResults []UpResult
	for _, up := range config.Ups {
		query := fmt.Sprintf(`min(up{namespace="%s"})`, up.Namespace)
		value, err := metricQuery(config.PrometheusServer, query)
		if err != nil {
			fatal(fmt.Sprintf("Could not make query %s: %s", query, err.Error()))
		}
		upResults = append(upResults, UpResult{
			Namespace: up.Namespace,
			Value:     int(value),
		})
	}

	longestNamespaceName := len("Namespace")
	for _, result := range upResults {
		if len(result.Namespace) > longestNamespaceName {
			longestNamespaceName = len(result.Namespace)
		}
	}
	lines = []string{fmt.Sprintf("%-*s  min(up)", longestNamespaceName, "Namespace")}
	for _, row := range upResults {
		lines = append(lines, fmt.Sprintf("%-*s  %d", longestNamespaceName, row.Namespace, row.Value))
	}
	fmt.Println(strings.Join(lines, "\n"))
}

func fatal(message string) {
	fmt.Fprintf(os.Stderr, "%s\n", message)
	os.Exit(1)
}

func metricQuery(server, query string) (float64, error) {
	query = url.QueryEscape(query)
	URL := fmt.Sprintf(queryURLFormat, server, query)
	resp, err := http.Get(URL)
	if err != nil {
		return 0, fmt.Errorf("Could not make query request: %s\n", err.Error())
	}
	if resp.StatusCode != 200 {
		return 0, fmt.Errorf("Got non-200 status: %s\n", resp.Status)
	}
	defer resp.Body.Close()
	var body QueryResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return 0, fmt.Errorf("Could not decode response body: %s\n", err.Error())
	}
	if len(body.Data.Result) == 0 {
		return 0.0, nil
	}
	value := body.Data.Result[0].Value[1].(string)
	numValue, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return 0, fmt.Errorf("Could not parse metric value into float: %s:", err.Error())
	}
	return numValue, nil
}

package collector

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/elotl/buildscaler/pkg/storage"
	"k8s.io/apimachinery/pkg/api/resource"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"
	"k8s.io/metrics/pkg/apis/external_metrics"
	"net/http"
	"net/http/httputil"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const (
	ScheduledJobsCount  = "ScheduledJobsCount"
	RunningJobsCount    = "RunningJobsCount"
	UnfinishedJobsCount = "UnfinishedJobsCount"
	WaitingJobsCount    = "WaitingJobsCount"
	IdleAgentCount      = "IdleAgentCount"
	BusyAgentCount      = "BusyAgentCount"
	TotalAgentCount     = "TotalAgentCount"
	BusyAgentPercentage = "BusyAgentPercentage"

	PollDurationHeader = `Buildkite-Agent-Metrics-Poll-Duration`
)

var (
	camel = regexp.MustCompile("(^[^A-Z0-9]*|[A-Z0-9]*)([A-Z0-9][^A-Z]+|$)")
)

type BuildkiteCollector struct {
	Endpoint  string
	Token     string
	UserAgent string
	Queues    []string
	Quiet     bool
	Debug     bool
	DebugHttp bool
	storage   *storage.ExternalMetricsMap
}

func NewBuildkiteCollector(storage *storage.ExternalMetricsMap, token, version string, queues []string) *BuildkiteCollector {
	return &BuildkiteCollector{
		Endpoint:  "https://agent.buildkite.com/v3", // should we pass it from flags?
		Token:     token,
		UserAgent: "elotl-buildscaler/" + version + " buildkite-metrics-collector",
		Queues:    queues,
		Quiet:     false,
		Debug:     false,
		DebugHttp: false,
		storage:   storage,
	}
}

func (c *BuildkiteCollector) Collect(cancel context.CancelFunc) error {
	r, err := c.collect()
	if err != nil {
		cancel()
		return err
	}
	for name, value := range r.Totals {
		key := fmt.Sprintf("buildkite_total_%s", camelToUnderscore(name))
		c.storage.OverrideOrStore(key, external_metrics.ExternalMetricValue{
			MetricName: key,
			Timestamp:  v1.NewTime(time.Now()),
			Value:      resource.MustParse(strconv.Itoa(value)),
		})
	}

	for queue, counts := range r.Queues {
		for name, value := range counts {
			key := fmt.Sprintf("buildkite_%s", camelToUnderscore(name))
			c.storage.OverrideOrStore(key, external_metrics.ExternalMetricValue{
				MetricName:   key,
				MetricLabels: map[string]string{"queue": queue},
				Timestamp:    v1.NewTime(time.Now()),
				Value:        resource.MustParse(strconv.Itoa(value)),
			})
		}
	}
	return nil
}

// Copyright (c) 2016 Buildkite Pty Ltd
// everything below is copied from buildkite/buildkite-agent-metrics
type Result struct {
	Totals       map[string]int
	Queues       map[string]map[string]int
	Org          string
	PollDuration time.Duration
}

type organizationResponse struct {
	Slug string `json:"slug"`
}

type metricsAgentsResponse struct {
	Idle  int `json:"idle"`
	Busy  int `json:"busy"`
	Total int `json:"total"`
}

type metricsJobsResponse struct {
	Scheduled int `json:"scheduled"`
	Running   int `json:"running"`
	Waiting   int `json:"waiting"`
	Total     int `json:"total"`
}

type queueMetricsResponse struct {
	Agents       metricsAgentsResponse `json:"agents"`
	Jobs         metricsJobsResponse   `json:"jobs"`
	Organization organizationResponse  `json:"organization"`
}

type allMetricsAgentsResponse struct {
	metricsAgentsResponse
	Queues map[string]metricsAgentsResponse `json:"queues"`
}

type allMetricsJobsResponse struct {
	metricsJobsResponse
	Queues map[string]metricsJobsResponse `json:"queues"`
}

type allMetricsResponse struct {
	Agents       allMetricsAgentsResponse `json:"agents"`
	Jobs         allMetricsJobsResponse   `json:"jobs"`
	Organization organizationResponse     `json:"organization"`
}

func (c *BuildkiteCollector) collect() (*Result, error) {
	result := &Result{
		Totals: map[string]int{},
		Queues: map[string]map[string]int{},
	}

	if len(c.Queues) == 0 {
		klog.V(5).Infof("Collecting agent metrics for all queues")

		endpoint, err := url.Parse(c.Endpoint)
		if err != nil {
			return nil, err
		}

		endpoint.Path += "/metrics"

		req, err := http.NewRequest("GET", endpoint.String(), nil)
		if err != nil {
			return nil, err
		}

		req.Header.Set("User-Agent", c.UserAgent)
		req.Header.Set("Authorization", fmt.Sprintf("Token %s", c.Token))

		if c.DebugHttp {
			if dump, err := httputil.DumpRequest(req, true); err == nil {
				klog.V(5).Infof("DEBUG request uri=%s\n%s\n", req.URL, dump)
			}
		}

		httpClient := &http.Client{
			Timeout: 15 * time.Second,
		}

		res, err := httpClient.Do(req)
		if err != nil {
			return nil, err
		}
		defer res.Body.Close()

		if c.DebugHttp {
			if dump, err := httputil.DumpResponse(res, true); err == nil {
				klog.V(5).Infof("DEBUG response uri=%s\n%s\n", req.URL, dump)
			}
		}

		// Handle any errors
		if res.StatusCode != http.StatusOK {
			// If it's json response, show the error message
			if strings.HasPrefix(res.Header.Get("Content-Type"), "application/json") {
				var errStruct struct {
					Message string `json:"message"`
				}
				err := json.NewDecoder(res.Body).Decode(&errStruct)
				if err == nil {
					return nil, errors.New(errStruct.Message)
				} else {
					klog.V(5).Infof("Failed to decode error: %v", err)
				}
			}

			return nil, fmt.Errorf("Request failed with %s (%d)", res.Status, res.StatusCode)
		}

		var allMetrics allMetricsResponse

		// Check if we get a poll duration header from server
		if pollSeconds := res.Header.Get(PollDurationHeader); pollSeconds != "" {
			pollSecondsInt, err := strconv.ParseInt(pollSeconds, 10, 64)
			if err != nil {
				klog.Infof("Failed to parse %s header: %v", PollDurationHeader, err)
			} else {
				result.PollDuration = time.Duration(pollSecondsInt) * time.Second
			}
		}

		err = json.NewDecoder(res.Body).Decode(&allMetrics)
		if err != nil {
			return nil, err
		}

		if allMetrics.Organization.Slug == "" {
			return nil, fmt.Errorf("No organization slug was found in the metrics response")
		}

		klog.Infof("Found organization %q", allMetrics.Organization.Slug)
		result.Org = allMetrics.Organization.Slug

		result.Totals[ScheduledJobsCount] = allMetrics.Jobs.Scheduled
		result.Totals[RunningJobsCount] = allMetrics.Jobs.Running
		result.Totals[UnfinishedJobsCount] = allMetrics.Jobs.Total
		result.Totals[WaitingJobsCount] = allMetrics.Jobs.Waiting
		result.Totals[IdleAgentCount] = allMetrics.Agents.Idle
		result.Totals[BusyAgentCount] = allMetrics.Agents.Busy
		result.Totals[TotalAgentCount] = allMetrics.Agents.Total
		result.Totals[BusyAgentPercentage] = busyAgentPercentage(allMetrics.Agents.metricsAgentsResponse)

		for queueName, queueJobMetrics := range allMetrics.Jobs.Queues {
			if _, ok := result.Queues[queueName]; !ok {
				result.Queues[queueName] = map[string]int{}
			}
			result.Queues[queueName][ScheduledJobsCount] = queueJobMetrics.Scheduled
			result.Queues[queueName][RunningJobsCount] = queueJobMetrics.Running
			result.Queues[queueName][UnfinishedJobsCount] = queueJobMetrics.Total
			result.Queues[queueName][WaitingJobsCount] = queueJobMetrics.Waiting
		}

		for queueName, queueAgentMetrics := range allMetrics.Agents.Queues {
			if _, ok := result.Queues[queueName]; !ok {
				result.Queues[queueName] = map[string]int{}
			}
			result.Queues[queueName][IdleAgentCount] = queueAgentMetrics.Idle
			result.Queues[queueName][BusyAgentCount] = queueAgentMetrics.Busy
			result.Queues[queueName][TotalAgentCount] = queueAgentMetrics.Total
			result.Queues[queueName][BusyAgentPercentage] = busyAgentPercentage(queueAgentMetrics)
		}
	} else {
		for _, queue := range c.Queues {
			klog.V(5).Infof("Collecting agent metrics for queue '%s'", queue)

			endpoint, err := url.Parse(c.Endpoint)
			if err != nil {
				return nil, err
			}

			endpoint.Path += "/metrics/queue"
			endpoint.RawQuery = url.Values{"name": {queue}}.Encode()

			req, err := http.NewRequest("GET", endpoint.String(), nil)
			if err != nil {
				return nil, err
			}

			req.Header.Set("User-Agent", c.UserAgent)
			req.Header.Set("Authorization", fmt.Sprintf("Token %s", c.Token))

			if c.DebugHttp {
				if dump, err := httputil.DumpRequest(req, true); err == nil {
					klog.V(5).Infof("DEBUG request uri=%s\n%s\n", req.URL, dump)
				}
			}

			res, err := http.DefaultClient.Do(req)
			if err != nil {
				return nil, err
			}

			if c.DebugHttp {
				if dump, err := httputil.DumpResponse(res, true); err == nil {
					klog.V(5).Infof("DEBUG response uri=%s\n%s\n", req.URL, dump)
				}
			}

			var queueMetrics queueMetricsResponse
			defer res.Body.Close()
			err = json.NewDecoder(res.Body).Decode(&queueMetrics)
			if err != nil {
				return nil, err
			}

			if queueMetrics.Organization.Slug == "" {
				return nil, fmt.Errorf("No organization slug was found in the metrics response")
			}

			klog.V(5).Infof("Found organization %q", queueMetrics.Organization.Slug)
			result.Org = queueMetrics.Organization.Slug

			result.Queues[queue] = map[string]int{
				ScheduledJobsCount:  queueMetrics.Jobs.Scheduled,
				RunningJobsCount:    queueMetrics.Jobs.Running,
				UnfinishedJobsCount: queueMetrics.Jobs.Total,
				WaitingJobsCount:    queueMetrics.Jobs.Waiting,
				IdleAgentCount:      queueMetrics.Agents.Idle,
				BusyAgentCount:      queueMetrics.Agents.Busy,
				TotalAgentCount:     queueMetrics.Agents.Total,
				BusyAgentPercentage: busyAgentPercentage(queueMetrics.Agents),
			}
		}
	}

	if !c.Quiet {
		result.Dump()
	}

	return result, nil
}

func busyAgentPercentage(agents metricsAgentsResponse) int {
	if agents.Total > 0 {
		return int(100 * agents.Busy / agents.Total)
	}
	return 0
}

func (r Result) Dump() {
	for name, c := range r.Totals {
		klog.V(5).Infof("Buildkite > Org=%s > %s=%d", r.Org, name, c)
	}

	for name, c := range r.Queues {
		for k, v := range c {
			klog.V(5).Infof("Buildkite > Org=%s > Queue=%s > %s=%d", r.Org, name, k, v)
		}
	}
}

func camelToUnderscore(s string) string {
	var a []string
	for _, sub := range camel.FindAllStringSubmatch(s, -1) {
		if sub[1] != "" {
			a = append(a, sub[1])
		}
		if sub[2] != "" {
			a = append(a, sub[2])
		}
	}
	return strings.ToLower(strings.Join(a, "_"))
}

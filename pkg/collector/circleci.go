package collector

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/elotl/buildscaler/pkg/storage"
	"k8s.io/apimachinery/pkg/api/resource"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/metrics/pkg/apis/external_metrics"
)

const (
	JobStatusFailed                = "failed"
	JobStatusRunning               = "running"
	JobStatusWaiting               = "waiting"
	ExternalMetricsJobsRunningName = "circleci_jobs_running"
	ExternalMetricsJobsWaitingName = "circleci_jobs_waiting"
	ExternalMetricsJobsFailedName  = "circleci_jobs_failed"
	CircleCIAPIEndpoint            = "https://circleci.com/api/v2"
)

type PaginatedResponse struct {
	NextPageToken string `json:"next_page_token"`
}

type ProjectPipeline struct {
	PipelineID string    `json:"id"`
	UpdatedAt  time.Time `json:"updated_at"`
	CreatedAt  time.Time `json:"created_at"`
}

type PaginatedProjectPipeline struct {
	PaginatedResponse `json:",inline"`
	Items             []ProjectPipeline `json:"items"`
}

type PipelineWorkflow struct {
	ID   string
	Name string
}

type PaginatedPipelineWorkflows struct {
	PaginatedResponse `json:",inline"`
	Items             []PipelineWorkflow `json:"items"`
}

type WorkflowJob struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Status string `json:"status"`
}

type PaginatedWorkflowJobs struct {
	PaginatedResponse `json:",inline"`
	Items             []WorkflowJob `json:"items"`
}

type WorkflowReport struct {
	JobsFailed  uint32
	JobsWaiting uint32
	JobsRunning uint32
}

type CircleCIClient struct {
	httpClient   http.Client
	endpoint     string
	pipelinesURL *url.URL
	token        string
}

func (cc *CircleCIClient) doRequest(req *http.Request, nextPageToken string) (*http.Response, error) {
	req.Header = make(map[string][]string)
	req.Header.Set("Circle-Token", cc.token)
	if nextPageToken != "" {
		q := req.URL.Query()
		q.Add("Circle-Token", nextPageToken)
		req.URL.RawQuery = q.Encode()
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	return cc.httpClient.Do(req)
}

func isPipelineTooOld(pipeline *ProjectPipeline, maxAge time.Duration) bool {
	ageThreshold := time.Now().Add(-maxAge)
	return pipeline.UpdatedAt.Before(ageThreshold)
}

func (cc *CircleCIClient) request() (*PaginatedProjectPipeline, error) {
	req := &http.Request{
		Method: "GET",
		URL:    cc.pipelinesURL,
	}
	resp, err := cc.doRequest(req, "")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var paginatedResp PaginatedProjectPipeline
	payload, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(payload, &paginatedResp)
	if err != nil {
		return nil, err
	}
	return &paginatedResp, nil
}

func (cc *CircleCIClient) listProjectPipelines(maxAge time.Duration) ([]ProjectPipeline, error) {
	projectPipelines := make([]ProjectPipeline, 0)

	var paginatedResp, err = cc.request()
	if err != nil {
		return nil, err
	}

	nextToken := paginatedResp.NextPageToken
	tooOld := false
	for _, pipeline := range paginatedResp.Items {
		if isPipelineTooOld(&pipeline, maxAge) {
			tooOld = true
			break
		}
		projectPipelines = append(projectPipelines, pipeline)
	}
	if tooOld {
		return projectPipelines, nil
	}
	for nextToken != "" {
		paginatedResp, err = cc.request()
		if err != nil {
			return nil, err
		}

		for _, pipeline := range paginatedResp.Items {
			if isPipelineTooOld(&pipeline, maxAge) {
				tooOld = true
				break
			}
			projectPipelines = append(projectPipelines, pipeline)
		}
		if tooOld {
			return projectPipelines, nil
		}

		nextToken = paginatedResp.NextPageToken
	}
	return projectPipelines, nil
}

func (cc *CircleCIClient) doListPipelineWorkflowsReq(workflowsURL *url.URL, nextToken string) (string, []PipelineWorkflow, error) {
	req := &http.Request{
		Method: "GET",
		URL:    workflowsURL,
	}
	resp, err := cc.doRequest(req, nextToken)
	if err != nil {
		return "", nil, err
	}
	var paginatedResp PaginatedPipelineWorkflows
	payload, err := ioutil.ReadAll(resp.Body)
	defer resp.Body.Close()
	if err != nil {
		return "", nil, err
	}
	err = json.Unmarshal(payload, &paginatedResp)
	if err != nil {
		return "", nil, err
	}
	return paginatedResp.NextPageToken, paginatedResp.Items, nil
}

func (cc *CircleCIClient) listPipelineWorkflows(pipelineID string) ([]PipelineWorkflow, error) {
	var pipelinesWorkflows []PipelineWorkflow
	workflowsURL, err := buildPipelineWorkflowsURL(cc.endpoint, pipelineID)
	if err != nil {
		return nil, err

	}
	nextToken, workflows, err := cc.doListPipelineWorkflowsReq(workflowsURL, "")
	if err != nil {
		return nil, err
	}
	pipelinesWorkflows = append(pipelinesWorkflows, workflows...)
	for nextToken != "" {
		newNextToken, workflows, err := cc.doListPipelineWorkflowsReq(workflowsURL, nextToken)
		if err != nil {
			return nil, err
		}
		pipelinesWorkflows = append(pipelinesWorkflows, workflows...)
		nextToken = newNextToken
	}
	return pipelinesWorkflows, nil
}

func (cc *CircleCIClient) doListWorkflowJobs(jobsURL *url.URL, nextToken string) (string, []WorkflowJob, error) {
	req := &http.Request{
		Method: "GET",
		URL:    jobsURL,
	}
	resp, err := cc.doRequest(req, "")
	if err != nil {
		return "", nil, err
	}
	var paginatedResp PaginatedWorkflowJobs
	payload, err := ioutil.ReadAll(resp.Body)
	defer resp.Body.Close()
	if err != nil {
		return "", nil, err
	}
	err = json.Unmarshal(payload, &paginatedResp)
	if err != nil {
		return "", nil, err
	}
	return paginatedResp.NextPageToken, paginatedResp.Items, nil
}

func (cc *CircleCIClient) listWorkflowJobs(workflowID string) ([]WorkflowJob, error) {
	jobsURL, err := BuildWorkflowJobsURL(cc.endpoint, workflowID)
	if err != nil {
		return nil, err
	}
	var jobs []WorkflowJob
	nextToken, workflowJobs, err := cc.doListWorkflowJobs(jobsURL, "")
	if err != nil {
		return nil, err
	}
	jobs = append(jobs, workflowJobs...)
	for nextToken != "" {
		newNextToken, workflowJobs, err := cc.doListWorkflowJobs(jobsURL, "")
		if err != nil {
			return nil, err
		}
		jobs = append(jobs, workflowJobs...)
		nextToken = newNextToken
	}
	return jobs, nil

}

type CircleCICollector struct {
	// maxPipelineAge allows us to filter out pipelines older than now - maxPipelineAge
	maxPipelineAge time.Duration
	client         *CircleCIClient
	projectSlug    string
	storage        *storage.ExternalMetricsMap
}

func buildProjectPipelinesURL(endpoint, projectSlug string) (*url.URL, error) {
	return url.Parse(endpoint + "/project/" + projectSlug + "/pipeline")
}

func buildPipelineWorkflowsURL(endpoint, pipelineID string) (*url.URL, error) {
	return url.Parse(endpoint + "/pipeline/" + pipelineID + "/workflow")
}

func BuildWorkflowJobsURL(endpoint, workflowID string) (*url.URL, error) {
	return url.Parse(endpoint + "/workflow/" + workflowID + "/job")
}

func NewCircleCICollector(token, projectSlug string, maxPipelineAge time.Duration, storage *storage.ExternalMetricsMap) (*CircleCICollector, error) {
	pipelinesURL, err := buildProjectPipelinesURL(CircleCIAPIEndpoint, projectSlug)
	if err != nil {
		return nil, err
	}
	// TODO: create a client here
	client := &CircleCIClient{
		httpClient: http.Client{
			Timeout: time.Second * 5,
		},
		token:        token,
		pipelinesURL: pipelinesURL,
	}
	return &CircleCICollector{client: client, maxPipelineAge: maxPipelineAge, projectSlug: projectSlug, storage: storage}, nil
}

func (c *CircleCICollector) Collect(cancel context.CancelFunc) error {
	// 1. Get a list of all pipelines in the project
	// 2. Filter only pipelines newer than now - maxPipelineAge
	// 3. Get all workflows for each pipeline
	// 4. Scrape list of workflow ids
	// 5. Loop over workflow ids and get all jobs for each workflow
	// 6. Calculate: Running / Pending jobs
	// 7. Store in c.storage as External Metrics
	pipelines, err := c.client.listProjectPipelines(c.maxPipelineAge)
	if err != nil {
		return err
	}
	var jobs []WorkflowJob
	var result WorkflowReport

	for _, pipeline := range pipelines {
		workflows, err := c.client.listPipelineWorkflows(pipeline.PipelineID)
		if err != nil {
			return err
		}
		for _, workflow := range workflows {
			workflowJobs, err := c.client.listWorkflowJobs(workflow.ID)
			if err != nil {
				return err
			}
			jobs = append(jobs, workflowJobs...)
		}
	}
	for _, job := range jobs {
		switch job.Status {
		case JobStatusFailed:
			result.JobsFailed++
		case JobStatusRunning:
			result.JobsRunning++
		case JobStatusWaiting:
			result.JobsWaiting++
		}
	}
	c.storage.OverrideOrStore(ExternalMetricsJobsRunningName, external_metrics.ExternalMetricValue{
		MetricName: ExternalMetricsJobsRunningName,
		MetricLabels: map[string]string{
			"project_slug": c.projectSlug,
		},
		Timestamp: v1.NewTime(time.Now()),
		Value:     resource.MustParse(strconv.Itoa(int(result.JobsRunning))),
	})
	c.storage.OverrideOrStore(ExternalMetricsJobsWaitingName, external_metrics.ExternalMetricValue{
		MetricName: ExternalMetricsJobsWaitingName,
		MetricLabels: map[string]string{
			"project_slug": c.projectSlug,
		},
		Timestamp: v1.NewTime(time.Now()),
		Value:     resource.MustParse(strconv.Itoa(int(result.JobsWaiting))),
	})
	c.storage.OverrideOrStore(ExternalMetricsJobsFailedName, external_metrics.ExternalMetricValue{
		MetricName: ExternalMetricsJobsFailedName,
		MetricLabels: map[string]string{
			"project_slug": c.projectSlug,
		},
		Timestamp: v1.NewTime(time.Now()),
		Value:     resource.MustParse(strconv.Itoa(int(result.JobsFailed))),
	})
	return nil
}

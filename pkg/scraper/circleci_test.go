package scraper

import (
	"context"
	"github.com/elotl/ciplatforms-external-metrics/pkg/storage"
	"github.com/stretchr/testify/assert"
	"io"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/metrics/pkg/apis/external_metrics"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync"
	"testing"
	"time"
)

func TestCircleCIScraper_Scrape(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/project/project-slug/pipeline":
			w.WriteHeader(http.StatusOK)
			_, _ = io.WriteString(w, `{
  "next_page_token": "dummy-token",
  "items": [
    {
      "id": "my-pipeline",
      "errors": [],
      "project_slug": "project-slug",
      "updated_at": "2021-08-25T09:13:15.550Z",
      "number": 23,
      "state": "created",
      "created_at": "2021-08-25T09:13:15.550Z"
    },
    {
      "id": "too-old-pipeline",
      "errors": [],
      "project_slug": "project-slug",
      "updated_at": "2000-08-25T09:11:49.733Z",
      "number": 22,
      "state": "created",
      "created_at": "2000-08-25T09:11:49.733Z"
    }
  ]
}
`)
		case "/pipeline/my-pipeline/workflow":
			w.WriteHeader(http.StatusOK)
			_, _ = io.WriteString(w, `{
  "next_page_token": null,
  "items": [
    {
      "pipeline_id": "my-pipeline",
      "id": "my-workflow",
      "name": "build",
      "project_slug": "project-slug",
      "status": "failed",
      "started_by": "744e3743-effc-4e6f-b127-9392386ad3cf",
      "pipeline_number": 21,
      "created_at": "2021-08-23T14:13:01Z",
      "stopped_at": "2021-08-23T14:47:49Z"
    }
  ]
}`)
		case "/workflow/my-workflow/job":
			w.WriteHeader(http.StatusOK)
			// 2 running jobs, 2 waiting, 1 failed
			_, _ = io.WriteString(w, `{
  "next_page_token": null,
  "items": [
    {
      "dependencies": [],
      "job_number": 112,
      "id": "08f2ac12-1881-4ce1-91b5-2b07fc14001c",
      "started_at": "2021-08-23T14:13:05Z",
      "name": "library-tests",
      "project_slug": "project-slug",
      "status": "running",
      "type": "build",
      "stopped_at": "2021-08-23T14:37:47Z"
    },
    {
      "dependencies": [],
      "job_number": 116,
      "id": "a9cb1218-dc42-4b21-8466-437fb24ea265",
      "started_at": "2021-08-23T14:13:06Z",
      "name": "ksapi-tests",
      "project_slug": "project-slug",
      "status": "running",
      "type": "build",
      "stopped_at": "2021-08-23T14:37:02Z"
    },
    {
      "dependencies": [],
      "job_number": 115,
      "id": "94ec8ca0-5071-43dc-8540-c01cb83bd941",
      "started_at": "2021-08-23T14:13:05Z",
      "name": "kickstarter-ui-tests",
      "project_slug": "project-slug",
      "status": "waiting",
      "type": "build",
      "stopped_at": "2021-08-23T14:36:01Z"
    },
    {
      "dependencies": [],
      "job_number": 113,
      "id": "6253e4b4-0f7f-46e8-b03e-04fba0a21f14",
      "started_at": "2021-08-23T14:13:05Z",
      "name": "kickstarter-tests",
      "project_slug": "project-slug",
      "status": "waiting",
      "type": "build",
      "stopped_at": "2021-08-23T14:45:44Z"
    },
    {
      "dependencies": [],
      "job_number": 114,
      "id": "62b8ba0b-eace-4257-9504-c00e600d2e89",
      "started_at": "2021-08-23T14:13:06Z",
      "name": "build-and-cache",
      "project_slug": "project-slug",
      "status": "failed",
      "type": "build",
      "stopped_at": "2021-08-23T14:36:07Z"
    }
  ]
}`)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	st := &storage.ExternalMetricsMap{
		RWMutex: &sync.RWMutex{},
		Data:    make(map[string]external_metrics.ExternalMetricValue),
	}
	pipelinesURL, err := url.Parse(s.URL + "/project/project-slug/pipeline")
	assert.NoError(t, err)
	client := &CircleCIClient{
		httpClient:   http.Client{},
		endpoint:     s.URL,
		pipelinesURL: pipelinesURL,
		token:        "dummy",
	}
	sc := &CircleCIScraper{
		maxPipelineAge: time.Hour * 24000,
		client:         client,
		projectSlug:    "project-slug",
		storage:        st,
	}
	err = sc.Scrape(context.CancelFunc(func() {}))
	assert.NoError(t, err)
	sc.storage.RWMutex.RLock()
	defer sc.storage.RWMutex.RUnlock()
	failedMetric := sc.storage.Data[ExternalMetricsJobsFailedName]
	runningMetric := sc.storage.Data[ExternalMetricsJobsRunningName]
	waitingMetric := sc.storage.Data[ExternalMetricsJobsWaitingName]
	assert.Equal(t, resource.MustParse("1"), failedMetric.Value)
	assert.Equal(t, resource.MustParse("2"), runningMetric.Value)
	assert.Equal(t, resource.MustParse("2"), waitingMetric.Value)

}

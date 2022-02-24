package collector

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"
	"k8s.io/metrics/pkg/apis/external_metrics"

	"github.com/elotl/buildscaler/pkg/storage"
)

type v1QueueInfo struct {
	OsFamily       string `json:"osFamily"`
	ContainerImage string `json:"containerImage"`
	Runner         int64  `json:"runners,string"`
	QueueSize      int64  `json:"queueSize,string"`
}

type v1QueueInfoDocument struct {
	QueueInfo []v1QueueInfo `json:"queueInfo"`
}

type Flarebuild struct {
	// https://api.stg.flare.build/api/v1
	client  *http.Client
	request *http.Request
	storage *storage.ExternalMetricsMap
}

func NewFlarebuild(storage *storage.ExternalMetricsMap, apiKey, endpoint string) (*Flarebuild, error) {
	var req, err = http.NewRequest("GET", endpoint+"/remote_executions/queues", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "buildscaler")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("x-api-key", apiKey)

	return &Flarebuild{
		request: req,
		client:  &http.Client{Timeout: 60 * time.Second},
		storage: storage,
	}, nil
}

func (c *Flarebuild) collect() (
	result []v1QueueInfo,
	err error,
) {
	var response *http.Response
	response, err = c.client.Do(c.request.Clone(context.TODO()))
	if err != nil {
		klog.Error("unable to query flare.build: %s", err)
		return
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		err = fmt.Errorf("bad http code: %d", response.StatusCode)
		return
	}
	var doc v1QueueInfoDocument
	if err = json.NewDecoder(response.Body).Decode(&doc); err != nil {
		return
	}
	result = doc.QueueInfo
	return
}

func flarebuildMetricName(os, name string) string {
	return fmt.Sprintf(
		"flarebuild_%s_%s", strings.ToLower(os), strings.ToLower(name),
	)
}

func flarebuildExternalMetricValue(
	os, image, name string, timestamp time.Time, value int64,
) *external_metrics.ExternalMetricValue {
	return &external_metrics.ExternalMetricValue{
		MetricName:   flarebuildMetricName(os, name),
		MetricLabels: map[string]string{"type": name, "os": os, "image": image},
		Timestamp:    metav1.NewTime(timestamp),
		Value:        *resource.NewQuantity(value, resource.DecimalSI),
	}
}

func (c *Flarebuild) Collect(cancel context.CancelFunc) error {
	var queues, err = c.collect()
	if err != nil {
		klog.ErrorS(err, "failed to collect data")
		cancel()
		return err
	}

	var now = time.Now()
	for _, q := range queues {
		var runner = flarebuildExternalMetricValue(q.OsFamily, q.ContainerImage, "runner", now, q.Runner)
		c.storage.OverrideOrStore(runner.MetricName, *runner)
		var queueSize = flarebuildExternalMetricValue(q.OsFamily, q.ContainerImage, "queue_size", now, q.QueueSize)
		c.storage.OverrideOrStore(queueSize.MetricName, *queueSize)
	}
	return nil
}

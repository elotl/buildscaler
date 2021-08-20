package ciprovider

import (
	"context"
	"errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/klog/v2"
	"k8s.io/metrics/pkg/apis/external_metrics"
	"sigs.k8s.io/custom-metrics-apiserver/pkg/provider"
	"sync"
)

type ExternalMetricsMap struct {
	RWMutex *sync.RWMutex
	Data map[string]external_metrics.ExternalMetricValue
}

type BuildkiteMetricsProvider struct {
	storage *ExternalMetricsMap
}

func NewBuildkiteMetricsProvider(storage *ExternalMetricsMap) *BuildkiteMetricsProvider {
	return &BuildkiteMetricsProvider{storage: storage}
}

type BuildkiteLabels struct {
	metric external_metrics.ExternalMetricValue
}

func (bl *BuildkiteLabels) Has(label string) (exists bool) {
	for k := range bl.metric.MetricLabels {
		if k == label {
			return true
		}
	}
	return false
}

func (bl *BuildkiteLabels) Get(label string) (value string) {
	for k, v := range bl.metric.MetricLabels {
		if k == label {
			return v
		}
	}
	return ""
}

func (b *BuildkiteMetricsProvider) GetExternalMetric(ctx context.Context, namespace string, metricSelector labels.Selector, info provider.ExternalMetricInfo) (*external_metrics.ExternalMetricValueList, error) {
	klog.V(6).Info("GetExternalMetric called with:")
	klog.V(6).Infof("ctx: %v namespace: %s metricSelector: %s info: %v", ctx, namespace, metricSelector, info.Metric)
	b.storage.RWMutex.RLock()
	val, ok := b.storage.Data[info.Metric]
	b.storage.RWMutex.RUnlock()
	if !ok {
		return nil, errors.New("metric " +  info.Metric + " not found")
	}
	externalMetric := val
	if metricSelector.Empty() {
		return &external_metrics.ExternalMetricValueList{
			Items: []external_metrics.ExternalMetricValue{externalMetric},
		}, nil
	}
	matcher := &BuildkiteLabels{metric: externalMetric}
	if !metricSelector.Matches(matcher) {
		return &external_metrics.ExternalMetricValueList{
			Items: []external_metrics.ExternalMetricValue{},
		}, errors.New("metric " +  info.Metric + " with labels " + metricSelector.String() + " not found")
	}
	return &external_metrics.ExternalMetricValueList{
		Items: []external_metrics.ExternalMetricValue{externalMetric},
	}, nil

}

func (b *BuildkiteMetricsProvider) ListAllExternalMetrics() []provider.ExternalMetricInfo {
	var (
		metrics []provider.ExternalMetricInfo
	)
	b.storage.RWMutex.RLock()
	defer b.storage.RWMutex.RUnlock()
	for key := range b.storage.Data {
		metrics = append(metrics, provider.ExternalMetricInfo{Metric: key})
	}
	klog.V(5).Infof("all external metrics: %s", metrics)
	return metrics
}

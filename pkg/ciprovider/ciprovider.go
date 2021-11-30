package ciprovider

import (
	"context"
	"errors"
	"github.com/elotl/buildscaler/pkg/storage"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/klog/v2"
	"k8s.io/metrics/pkg/apis/external_metrics"
	"sigs.k8s.io/custom-metrics-apiserver/pkg/provider"
)

type ExternalMetricsProviderFromStorage struct {
	storage *storage.ExternalMetricsMap
}

func NewExternalMetricsProviderFromStorage(storage *storage.ExternalMetricsMap) *ExternalMetricsProviderFromStorage {
	return &ExternalMetricsProviderFromStorage{storage: storage}
}

type ExternalMetricsLabelsMatcher struct {
	metric external_metrics.ExternalMetricValue
}

func (em *ExternalMetricsLabelsMatcher) Has(label string) (exists bool) {
	for k := range em.metric.MetricLabels {
		if k == label {
			return true
		}
	}
	return false
}

func (em *ExternalMetricsLabelsMatcher) Get(label string) (value string) {
	for k, v := range em.metric.MetricLabels {
		if k == label {
			return v
		}
	}
	return ""
}

func (ep *ExternalMetricsProviderFromStorage) GetExternalMetric(ctx context.Context, namespace string, metricSelector labels.Selector, info provider.ExternalMetricInfo) (*external_metrics.ExternalMetricValueList, error) {
	klog.V(6).Info("GetExternalMetric called with:")
	klog.V(6).Infof("ctx: %v namespace: %s metricSelector: %s info: %v", ctx, namespace, metricSelector, info.Metric)
	ep.storage.RWMutex.RLock()
	val, ok := ep.storage.Data[info.Metric]
	ep.storage.RWMutex.RUnlock()
	if !ok {
		return nil, errors.New("metric " + info.Metric + " not found")
	}
	externalMetric := val
	if metricSelector.Empty() {
		return &external_metrics.ExternalMetricValueList{
			Items: []external_metrics.ExternalMetricValue{externalMetric},
		}, nil
	}
	matcher := &ExternalMetricsLabelsMatcher{metric: externalMetric}
	if !metricSelector.Matches(matcher) {
		return &external_metrics.ExternalMetricValueList{
			Items: []external_metrics.ExternalMetricValue{},
		}, errors.New("metric " + info.Metric + " with labels " + metricSelector.String() + " not found")
	}
	return &external_metrics.ExternalMetricValueList{
		Items: []external_metrics.ExternalMetricValue{externalMetric},
	}, nil

}

func (ep *ExternalMetricsProviderFromStorage) ListAllExternalMetrics() []provider.ExternalMetricInfo {
	return ep.storage.ListExternalMetricInfo()
}

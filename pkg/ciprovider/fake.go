package ciprovider

import (
	"context"
	"errors"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/klog/v2"
	"k8s.io/metrics/pkg/apis/external_metrics"
	"sigs.k8s.io/custom-metrics-apiserver/pkg/provider"
	"strconv"
	"sync"
)

type FakeProvider struct {
	storage *sync.Map
}

func NewFakeProvider(storage *sync.Map) *FakeProvider {
	return &FakeProvider{storage: storage}
}

func (fk *FakeProvider) GetExternalMetric(ctx context.Context, namespace string, metricSelector labels.Selector, info provider.ExternalMetricInfo) (*external_metrics.ExternalMetricValueList, error) {
	klog.V(5).Info("GetExternalMetric called with:")
	klog.V(5).Infof("ctx: %v namespace: %s metricSelector: %s info: %v", ctx, namespace, metricSelector, info.Metric)
	klog.V(5).Infof("trying to load metric %s from storage", info.Metric)
	metric, ok := fk.storage.Load(info.Metric)
	if !ok {
		return nil, errors.New("cannot find metric")
	}
	klog.V(5).Infof("got %v value, trying to cast it to string", metric)
	value := strconv.Itoa(metric.(int))
	klog.V(5).Infof("casted to %s", value)
	return &external_metrics.ExternalMetricValueList{
		Items: []external_metrics.ExternalMetricValue{
			{
				MetricName: "build_queue_waiting",
				Value:      resource.MustParse(value),
			},
		},
	}, nil
}

func (fk *FakeProvider) ListAllExternalMetrics() []provider.ExternalMetricInfo {
	return []provider.ExternalMetricInfo{
		{
			Metric: "build_queue_waiting",
		},
	}
}

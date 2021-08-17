package ciprovider

import (
	"context"
	"errors"
	"k8s.io/apimachinery/pkg/api/resource"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/metrics/pkg/apis/external_metrics"
	"sigs.k8s.io/custom-metrics-apiserver/pkg/provider"
	"sync"
)

type FakeProvider struct {
	storage *sync.Map
}

func NewFakeProvider(storage *sync.Map) *FakeProvider {
	return &FakeProvider{storage: storage}
}

func (fk *FakeProvider) GetExternalMetric(ctx context.Context, namespace string, metricSelector labels.Selector, info provider.ExternalMetricInfo) (*external_metrics.ExternalMetricValueList, error) {
	metric, ok := fk.storage.Load(info.Metric)
	if !ok {
		return nil, errors.New("cannot find metric")
	}
	value := metric.(string)
	return &external_metrics.ExternalMetricValueList{
		TypeMeta: v1.TypeMeta{},
		ListMeta: v1.ListMeta{},
		Items: []external_metrics.ExternalMetricValue{
			{
				MetricName:   "build_queue_waiting",
				MetricLabels: map[string]string{},
				Value:        resource.MustParse(value),
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

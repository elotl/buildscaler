package ciprovider

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/elotl/buildscaler/pkg/storage"
	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/metrics/pkg/apis/external_metrics"
	"sigs.k8s.io/custom-metrics-apiserver/pkg/provider"
)

func TestExternalMetricsProviderFromStorage_GetExternalMetric(t *testing.T) {
	cases := []struct {
		name           string
		data           map[string]external_metrics.ExternalMetricValue
		metricSelector labels.Selector
		info           provider.ExternalMetricInfo
		expectedList   *external_metrics.ExternalMetricValueList
		expectedErr    error
	}{
		{
			name: "by_name",
			data: map[string]external_metrics.ExternalMetricValue{
				"metric1": {
					MetricName: "metric1",
					Value:      resource.MustParse("42"),
				},
				"not-my-metric": {
					MetricName: "not-my-metric",
					Value:      resource.MustParse("1"),
				},
			},
			metricSelector: labels.NewSelector(),
			info:           provider.ExternalMetricInfo{Metric: "metric1"},
			expectedList: &external_metrics.ExternalMetricValueList{
				Items: []external_metrics.ExternalMetricValue{
					{
						MetricName: "metric1",
						Value:      resource.MustParse("42"),
					},
				},
			},
			expectedErr: nil,
		},
		{
			name: "by_name_and_label",
			data: map[string]external_metrics.ExternalMetricValue{
				"metric1": {
					MetricName: "metric1",
					Value:      resource.MustParse("42"),
					MetricLabels: map[string]string{
						"label-key": "label-val",
					},
				},
				"not-my-metric": {
					MetricName: "not-my-metric",
					Value:      resource.MustParse("1"),
				},
			},
			metricSelector: labels.SelectorFromValidatedSet(map[string]string{"label-key": "label-val"}),
			info:           provider.ExternalMetricInfo{Metric: "metric1"},
			expectedList: &external_metrics.ExternalMetricValueList{
				Items: []external_metrics.ExternalMetricValue{
					{
						MetricName:   "metric1",
						Value:        resource.MustParse("42"),
						MetricLabels: map[string]string{"label-key": "label-val"},
					},
				},
			},
			expectedErr: nil,
		},
		{
			name: "by_name_and_label_does_not_match",
			data: map[string]external_metrics.ExternalMetricValue{
				"metric1": {
					MetricName: "metric1",
					Value:      resource.MustParse("42"),
					MetricLabels: map[string]string{
						"label-key": "label-val",
					},
				},
				"not-my-metric": {
					MetricName: "not-my-metric",
					Value:      resource.MustParse("1"),
				},
			},
			metricSelector: labels.SelectorFromValidatedSet(map[string]string{"label-key": "NOT-label-val"}),
			info:           provider.ExternalMetricInfo{Metric: "metric1"},
			expectedList: &external_metrics.ExternalMetricValueList{
				Items: []external_metrics.ExternalMetricValue{},
			},
			expectedErr: errors.New("metric metric1 with labels label-key=NOT-label-val not found"),
		},
		{
			name: "not_found",
			data: map[string]external_metrics.ExternalMetricValue{
				"not-my-metric": {
					MetricName: "not-my-metric",
					Value:      resource.MustParse("1"),
				},
			},
			metricSelector: labels.NewSelector(),
			info:           provider.ExternalMetricInfo{Metric: "metric1"},
			expectedList:   nil,
			expectedErr:    errors.New("metric metric1 not found"),
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			st := &storage.ExternalMetricsMap{
				RWMutex: &sync.RWMutex{},
				Data:    tc.data,
			}
			metricProvider := NewExternalMetricsProviderFromStorage(st)
			got, err := metricProvider.GetExternalMetric(context.TODO(), "", tc.metricSelector, tc.info)
			assert.Equal(t, tc.expectedErr, err)
			assert.Equal(t, tc.expectedList, got)
		})
	}
}

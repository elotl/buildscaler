package storage

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/metrics/pkg/apis/external_metrics"
	"sigs.k8s.io/custom-metrics-apiserver/pkg/provider"
)

func TestExternalMetricsMap_OverrideOrStore(t *testing.T) {
	cases := []struct {
		name         string
		initialData  map[string]external_metrics.ExternalMetricValue
		key          string
		val          external_metrics.ExternalMetricValue
		expectedData map[string]external_metrics.ExternalMetricValue
	}{
		{
			name:        "store",
			initialData: make(map[string]external_metrics.ExternalMetricValue),
			key:         "metric",
			val: external_metrics.ExternalMetricValue{
				MetricName: "metric",
				Value:      resource.MustParse("1"),
			},
			expectedData: map[string]external_metrics.ExternalMetricValue{
				"metric": {
					MetricName: "metric",
					Value:      resource.MustParse("1"),
				},
			},
		},
		{
			name: "override",
			initialData: map[string]external_metrics.ExternalMetricValue{
				"metric": {
					MetricName: "metric",
					Value:      resource.MustParse("1"),
				},
			},
			key: "metric",
			val: external_metrics.ExternalMetricValue{
				MetricName: "metric",
				Value:      resource.MustParse("2"),
			},
			expectedData: map[string]external_metrics.ExternalMetricValue{
				"metric": {
					MetricName: "metric",
					Value:      resource.MustParse("2"),
				},
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rwm := &sync.RWMutex{}
			st := &ExternalMetricsMap{
				RWMutex: rwm,
				Data:    tc.initialData,
			}
			st.OverrideOrStore(tc.key, tc.val)
			assert.Equal(t, tc.expectedData, st.Data)
		})
	}
}

func TestExternalMetricsMap_ListExternalMetricInfo(t *testing.T) {
	cases := []struct {
		name        string
		initialData map[string]external_metrics.ExternalMetricValue
		expected    []provider.ExternalMetricInfo
	}{
		{
			name: "happy path",
			initialData: map[string]external_metrics.ExternalMetricValue{
				"metric1": {},
				"metric2": {},
			},
			expected: []provider.ExternalMetricInfo{
				{
					Metric: "metric1",
				},
				{
					Metric: "metric2",
				},
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rwm := &sync.RWMutex{}
			st := &ExternalMetricsMap{
				RWMutex: rwm,
				Data:    tc.initialData,
			}
			got := st.ListExternalMetricInfo()
			for _, metric := range got {
				assert.Contains(t, tc.expected, metric)
			}
		})
	}
}

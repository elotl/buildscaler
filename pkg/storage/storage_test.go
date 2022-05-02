/*
Copyright 2022 Elotl Inc.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
    http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

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

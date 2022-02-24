package storage

import (
	"sync"

	"k8s.io/klog/v2"
	"k8s.io/metrics/pkg/apis/external_metrics"
	"sigs.k8s.io/custom-metrics-apiserver/pkg/provider"
)

type ExternalMetricsMap struct {
	RWMutex *sync.RWMutex
	Data    map[string]external_metrics.ExternalMetricValue
}

func NewExternalMetricsMap() *ExternalMetricsMap {
	return &ExternalMetricsMap{
		RWMutex: &sync.RWMutex{},
		Data:    make(map[string]external_metrics.ExternalMetricValue),
	}
}

func (e *ExternalMetricsMap) OverrideOrStore(key string, value external_metrics.ExternalMetricValue) {
	e.RWMutex.Lock()
	defer e.RWMutex.Unlock()
	_, ok := e.Data[key]
	if ok {
		klog.V(5).Infof("metric %s already has value, overwriting...", key)
		delete(e.Data, key)

	}
	e.Data[key] = value
	klog.V(5).Infof("metric %s successfully scraped and stored.", key)
}

func (e *ExternalMetricsMap) ListExternalMetricInfo() []provider.ExternalMetricInfo {
	metrics := make([]provider.ExternalMetricInfo, 0, len(e.Data))
	e.RWMutex.RLock()
	defer e.RWMutex.RUnlock()
	for key := range e.Data {
		metrics = append(metrics, provider.ExternalMetricInfo{Metric: key})
	}
	klog.V(5).Infof("all external metrics: %s", metrics)
	return metrics
}

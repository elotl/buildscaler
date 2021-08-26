package storage

import (
	"k8s.io/klog/v2"
	"k8s.io/metrics/pkg/apis/external_metrics"
	"sigs.k8s.io/custom-metrics-apiserver/pkg/provider"
	"sync"
)

type ExternalMetricsMap struct {
	RWMutex *sync.RWMutex
	Data    map[string]external_metrics.ExternalMetricValue
}

func (e *ExternalMetricsMap) OverrideOrStore(key string, value external_metrics.ExternalMetricValue) {
	e.RWMutex.RLock()
	_, ok := e.Data[key]
	e.RWMutex.RUnlock()
	if ok {
		klog.V(5).Infof("metric %s already has value, overwriting...", key)
		e.RWMutex.Lock()
		delete(e.Data, key)
		e.RWMutex.Unlock()

	}
	e.RWMutex.Lock()
	e.Data[key] = value
	e.RWMutex.Unlock()
	klog.V(5).Infof("metric %s successfully scraped and stored.", key)
}

func (e *ExternalMetricsMap) ListExternalMetricInfo() []provider.ExternalMetricInfo {
	var metrics []provider.ExternalMetricInfo
	e.RWMutex.RLock()
	defer e.RWMutex.RUnlock()
	for key := range e.Data {
		metrics = append(metrics, provider.ExternalMetricInfo{Metric: key})
	}
	klog.V(5).Infof("all external metrics: %s", metrics)
	return metrics
}

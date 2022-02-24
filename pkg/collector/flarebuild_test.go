package collector

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/api/resource"

	"github.com/elotl/buildscaler/pkg/storage"
)

func TestFlarebuildBasic(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, r.URL.Path, "/remote_executions/queues")
		assert.Equal(t, r.Header["User-Agent"], []string{"buildscaler"})
		assert.Equal(t, r.Header["Accept"], []string{"application/json"})
		assert.Equal(t, r.Header["X-Api-Key"], []string{"fakeauth"})
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, `{"queueInfo": [
{"osFamily": "MacOS", "containerImage": "", "runners": "9", "queueSize": "6"},
{"osFamily": "Linux", "containerImage": "docker://gcr.io/flare-build-alpha/u", "runners": "4", "queueSize": "1"}
]}
`)
	}))
	defer s.Close()

	store := storage.NewExternalMetricsMap()
	fb, err := NewFlarebuild(store, "fakeauth", s.URL)
	assert.Nil(t, err)
	err = fb.Collect(func() {})
	assert.Nil(t, err)

	var m = store.Data["flarebuild_macos_runner"]
	assert.Equal(t, "flarebuild_macos_runner", m.MetricName)
	assert.Equal(
		t,
		map[string]string{"os": "MacOS", "image": "", "type": "runner"},
		m.MetricLabels)
	assert.Equal(t, *resource.NewQuantity(9, resource.DecimalSI), m.Value)

	m = store.Data["flarebuild_macos_queue_size"]
	assert.Equal(t, "flarebuild_macos_queue_size", m.MetricName)
	assert.Equal(
		t,
		map[string]string{"os": "MacOS", "image": "", "type": "queue_size"},
		m.MetricLabels)
	assert.Equal(t, *resource.NewQuantity(6, resource.DecimalSI), m.Value)

	m = store.Data["flarebuild_linux_runner"]
	assert.Equal(t, "flarebuild_linux_runner", m.MetricName)
	assert.Equal(
		t,
		map[string]string{"os": "Linux", "image": "docker://gcr.io/flare-build-alpha/u", "type": "runner"},
		m.MetricLabels)
	assert.Equal(t, *resource.NewQuantity(4, resource.DecimalSI), m.Value)

	m = store.Data["flarebuild_linux_queue_size"]
	assert.Equal(t, "flarebuild_linux_queue_size", m.MetricName)
	assert.Equal(
		t,
		map[string]string{"os": "Linux", "image": "docker://gcr.io/flare-build-alpha/u", "type": "queue_size"},
		m.MetricLabels)
	assert.Equal(t, *resource.NewQuantity(1, resource.DecimalSI), m.Value)
}

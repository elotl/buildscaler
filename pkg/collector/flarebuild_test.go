package collector

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/metrics/pkg/apis/external_metrics"

	"github.com/elotl/buildscaler/pkg/storage"
)

func TestFlarebuildBasic(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, r.URL.Path, "/remote_executions/queues")
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, `{"queueInfo": [{
"osFamily": "MacOS", "containerImage": "", "runners": "4", "queueSize": "0"
},
{
"osFamily": "Linux", "containerImage": "docker://gcr.io/flare-build-alpha/ubuntu2004-rbe@sha256:24dd801f4eea130bb4739893fffecd227df8378db1712efff00e4d5fd8968006", "runners": "4", "queueSize": "0"
}
]
}
`)
	}))
	defer s.Close()

	store := storage.NewExternalMetricsMap()
	fb, err := NewFlarebuild(store, "fakeauth", s.URL)
	assert.Nil(t, err)
	err = fb.Collect(func() {})
	assert.Nil(t, err)
	assert.Equal(t, store.Data, map[string]external_metrics.ExternalMetricValue{
		"macos_runner": {
			MetricName: "macos_runner",
			Value:      resource.MustParse("4"),
		},
	})
}

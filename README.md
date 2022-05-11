[![Build and Test](https://github.com/elotl/buildscaler/actions/workflows/build.yml/badge.svg)](https://github.com/elotl/buildscaler/actions/workflows/build.yml)
[![Lint](https://github.com/elotl/buildscaler/actions/workflows/lint.yml/badge.svg)](https://github.com/elotl/buildscaler/actions/workflows/lint.yml)
# Buildscaler

Built using [customer-metrics
lib](https://github.com/kubernetes-sigs/custom-metrics-apiserver) A service
meant to provide k8s External Metrics from CI providers API, which can be
later used to configure autoscaling via Horizontal Pod Autoscalers.

# Requirements

[goimports](https://pkg.go.dev/golang.org/x/tools/cmd/goimports) &
[staticcheck](https://staticcheck.io/docs/):

    $ go get golang.org/x/tools/cmd/goimports
    $ go install honnef.co/go/tools/cmd/staticcheck@latest

# Supported providers

## Buildkite

You have to set up `BUILDKITE_AGENT_TOKEN` env variable.

Exported metrics (list of metrics can be queried via kubectl `$ kubectl get --raw="/apis/external.metrics.k8s.io/v1beta1/" -A  | jq -r ".resources[].name" | sort`
Querying specific metric for its value: 
`$ kubectl get --raw="/apis/external.metrics.k8s.io/v1beta1/namespaces/external-metrics/buildkite_waiting_jobs_count" -A  | jq` 
```bash{
  "kind": "ExternalMetricValueList",
  "apiVersion": "external.metrics.k8s.io/v1beta1",
  "metadata": {},
  "items": [
    {
      "metricName": "buildkite_waiting_jobs_count",
      "metricLabels": {
        "queue": "macos"
      },
      "timestamp": "2021-08-30T12:44:30Z",
      "value": "0"
    }
  ]
}
```

| Metric name                     | Description |
|---------------------------------|-------------|
| buildkite_busy_agent_count      | ...         |
| buildkite_busy_agent_percentage | ...         |
| buildkite_idle_agent_count      | ...         |
| buildkite_running_jobs_count    | ...         |
| buildkite_scheduled_jobs_count  | ...         |
| buildkite_total_agent_count     | ...         |
| buildkite_unfinished_jobs_count | ...         |
| buildkite_waiting_jobs_count    | ...         |

Scraper provides Buildkite queue tag as a label for each metric.

### Installation for Buildkite

1. Set `BUILDKITE_AGENT_TOKEN` environment variable.
2. Create external-metrics namespace in your cluster `kubectl create namespace external-metrics`
3. Create a secret with your BUILDKITE_AGENT_TOKEN: `kubectl create secret generic --namespace=external-metrics buildkite-secrets --from-literal=BUILDKITE_AGENT_TOKEN=$BUILDKITE_AGENT_TOKEN`


    $ kubectl apply -f deploy/rbac.yaml
      kubectl apply -f deploy/rbac-kube-system.yaml
      kubectl apply -f deploy/builldkite-deployment.yaml
      kubectl apply -f deploy/service.yaml
      kubectl apply -f deploy/apiservice.yaml

## CircleCI

You have to set up `CIRCLECI_TOKEN` & `CIRCLECI_PROJECT_SLUG`. Currently,
CircleCI scraper supports only 1 project at the time, meaning that you need
to have metrics about more projects, you need to run multiple deployments of
`buildscaler` with different env vars.  Scraper takes into account only jobs
from pipelines updated earlier than 30 minutes ago.

Exported metrics:

| Metric name           | Description                |
|-----------------------|----------------------------|
| circleci_jobs_failed  | jobs with status "failed"  |
| circleci_jobs_running | jobs with status "running" |
| circleci_jobs_waiting | jobs with status "waiting" |

## Flare.build

Set up `FLAREBUILD_API_KEY`. One metric per os/image combo. There may therefor
be multiple flarebuild_linux_runners metrics due to multiple linux images begin
run.

Exported metrics:

| Metric name                  | Description                               |
|------------------------------|-------------------------------------------|
| `flarebuild_<os>_runners`    | Number of runners for this os/image combo |
| `flarebuild_<os>_queue_size` | Queue size for this os/image combo        |


# Deployment

1. Edit a following lines in [deployment.yaml](deploy/deployment.yaml): ` --ci-platform=circleci` <- set to buildkite/circleci
2. Add environment variables mentioned in a section above about your CI provider.
3. Run `kubectl apply -f deploy/*.yaml`
# CI Platforms External Metrics

Built using [customer-metrics lib](https://github.com/kubernetes-sigs/custom-metrics-apiserver)
A service meant to provide k8s External Metrics from CI providers API, which can be later used to configure autoscaling via Horizontal Pod Autoscalers.

## Supported providers

### Buildkite
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

| Metric name | Description |
| ----------- | ----------- |
| buildkite_busy_agent_count | ... |
| buildkite_busy_agent_percentage | ... |
| buildkite_idle_agent_count | ... |
| buildkite_running_jobs_count | ... |
| buildkite_scheduled_jobs_count | ... |
| buildkite_total_agent_count | ... |
| buildkite_unfinished_jobs_count | ... |
| buildkite_waiting_jobs_count | ... |

Scraper provides Buildkite queue tag as a label for each metric.

2. CircleCI
You have to set up `CIRCLECI_TOKEN` & `CIRCLECI_PROJECT_SLUG`. Currently, CircleCI scraper supports only 1 project at the time, meaning that you need to have metrics about more projects, you need to run multiple deployments of `ciplatforms-external-metrics` with different env vars.
Scraper takes into account only jobs from pipelines updated earlier than 30 minutes ago.

Exported metrics:

| Metric name | Description |
| ----------- | ----------- |
| circleci_jobs_failed | jobs with status "failed" |
| circleci_jobs_running | jobs with status "running" |
| circleci_jobs_waiting | jobs with status "waiting" |



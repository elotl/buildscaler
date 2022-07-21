[![Build and Test](https://github.com/elotl/buildscaler/actions/workflows/build.yml/badge.svg)](https://github.com/elotl/buildscaler/actions/workflows/build.yml)
[![Lint](https://github.com/elotl/buildscaler/actions/workflows/lint.yml/badge.svg)](https://github.com/elotl/buildscaler/actions/workflows/lint.yml)
# Buildscaler

Built using [customer-metrics
lib](https://github.com/kubernetes-sigs/custom-metrics-apiserver) A service
meant to provide k8s External Metrics from CI providers API, which can be
later used to configure autoscaling via Horizontal Pod Autoscalers.

# Requirements

[golangci-lint](https://golangci-lint.run/)

    $ go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.45.2

# Supported providers

## Buildkite

### Install

Requirements:

1. kubectl pointed at the cluster you want to deploy buildscaler on
2. A namespace where you’ll deploy buildscaler. You can create it like this
   `kubectl create namespace external-metrics`.

Before deploying Buildscaler, create a secret named
`buildkite-agent` with a `token` entry, and put the Buildkite Agent Token in
it. You can retreive the Buildkite agent’s token from the organization’s agents
page at [builkite.com](https://buildkite.com/).

    $ kubectl create secret generic \
        buildkite-agent \
        --from-literal=token=$BUILDKITE_AGENT_TOKEN \
        --namespace=$NAMESPACE

Then run the install script to deploy buildscaler in your cluster:

	$ cd deploy; ./deploy.sh "$NAMESPACE"

### Usage for Buildkite

You can use our [manifests](examples/buildkite) as a good starting point for deploying your Buildkite Agent deployment and using exported metrics in HorizontalPodAutoscaler.

List of exported metrics:

| Metric name                     | Description                       |
|---------------------------------|-----------------------------------|
| buildkite_busy_agent_count      | Number of busy agents             |
| buildkite_busy_agent_percentage | Percentage of busy agents         |
| buildkite_idle_agent_count      | Number of idle agents             |
| buildkite_running_jobs_count    | Number of currently running jobs  |
| buildkite_scheduled_jobs_count  | Number of scheduled jobs          |
| buildkite_total_agent_count     | Total number of agents connected  |
| buildkite_unfinished_jobs_count | Number of unfinished jobs.        |
| buildkite_waiting_jobs_count    | Number of jobs waiting in a queue |

Scraper provides Buildkite queue tag as a label for each metric.

To get a list of exported metrics, you can use following kubectl command:

    $ kubectl get --raw="/apis/external.metrics.k8s.io/v1beta1/" -A  | jq -r ".resources[].name" | sort

To get specific metric details:

    $ kubectl get --raw="/apis/external.metrics.k8s.io/v1beta1/namespaces/external-metrics/buildkite_waiting_jobs_count" -A  | jq
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

Additional information about data exposed by Buildkite can be found [here](https://buildkite.com/docs/apis/agent-api/metrics). Buildscaler is using https://agent.buildkite.com/v3/metrics endpoint as a data source.

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

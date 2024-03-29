[![Build and Test](https://github.com/elotl/buildscaler/actions/workflows/build.yml/badge.svg)](https://github.com/elotl/buildscaler/actions/workflows/build.yml)
[![Lint](https://github.com/elotl/buildscaler/actions/workflows/lint.yml/badge.svg)](https://github.com/elotl/buildscaler/actions/workflows/lint.yml)
# Buildscaler

Built using [customer-metrics
lib](https://github.com/kubernetes-sigs/custom-metrics-apiserver) A service
meant to provide k8s External Metrics from CI providers API, which can be
later used to configure autoscaling via Horizontal Pod Autoscalers.

# Requirements

[kubectl](https://kubernetes.io/docs/tasks/tools/#kubectl)

# Buildkite

## Install

Requirements:

1. kubectl pointed at the cluster you want to deploy buildscaler on
2. A namespace where you’ll deploy buildscaler. You can create it like this
   `kubectl create namespace $NAMESPACE`.

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

By default all the metrics from all the Buildkite queues will be reported,
if you wish to only report metrics for a subset of the queues, you can set
the environment variable `BUILDKITE_QUEUES` in Buildscaler’s deployment:

```
    env:
      - name: BUILDKITE_AGENT_TOKEN
        valueFrom:
          secretKeyRef:
            name: buildkite-agent
            key: token
      - name: BUILDKITE_QUEUES
        value: queue1,queue2
```

## Usage

You can use our [manifests](examples/buildkite) as a good starting point for deploying your Buildkite Agent deployment and using exported metrics in HorizontalPodAutoscaler.

List of exported metrics:

| Metric name                     | Description                       |
|---------------------------------|-----------------------------------|
| buildkite_totla_busy_agent_count      | Number of busy agents             |
| buildkite_totla_busy_agent_percentage | Percentage of busy agents         |
| buildkite_totla_idle_agent_count      | Number of idle agents             |
| buildkite_totla_running_jobs_count    | Number of currently running jobs  |
| buildkite_totla_scheduled_jobs_count  | Number of scheduled jobs          |
| buildkite_totla_total_agent_count     | Total number of agents connected  |
| buildkite_totla_unfinished_jobs_count | Number of unfinished jobs.        |
| buildkite_totla_waiting_jobs_count    | Number of jobs waiting in a queue |

Scraper provides Buildkite queue tag as a label for each metric.

To get a list of exported metrics, you can use following kubectl command:

    $ kubectl get --raw="/apis/external.metrics.k8s.io/v1beta1/" -A  | jq -r ".resources[].name" | sort

To get specific metric details:

    $ kubectl get --raw="/apis/external.metrics.k8s.io/v1beta1/namespaces/$NAMESPACE/buildkite_waiting_jobs_count" -A  | jq
```bash{
  "kind": "ExternalMetricValueList",
  "apiVersion": "external.metrics.k8s.io/v1beta1",
  "metadata": {},
  "items": [
    {
      "metricName": "buildkite_total_waiting_jobs_count",
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

# CircleCI

You can re-use the Buildkite deployment and switch to the CircleCI provider
by passing the flag `-ci-platform=circleci` to the buildscaler command.

The environment variables `CIRCLECI_TOKEN` and `CIRCLECI_PROJECT_SLUG`
are required.

Currently, the CircleCI scraper supports a single project. Therefor if you
require metrics for multiple projects you will have to run multiple instances
of Buildcaler: one per project.

The scraper reports metrics for jobs updated in the past 30 minutes. If a
pipeline’s last job update is more than 30 minutes old it will be ignored.

Exported metrics:

| Metric name           | Description                |
|-----------------------|----------------------------|
| circleci_jobs_failed  | jobs with status "failed"  |
| circleci_jobs_running | jobs with status "running" |
| circleci_jobs_waiting | jobs with status "waiting" |

# Flare.build

You can re-use the Buildkite deployment and switch to the CircleCI provider
by passing the flag `-ci-platform=flarebuild` to the buildscaler command.

Set the `FLAREBUILD_API_KEY`. One metric per os/image combo. There may therefor
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

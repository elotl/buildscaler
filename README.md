# CI Platforms External Metrics

Built using [customer-metrics lib](https://github.com/kubernetes-sigs/custom-metrics-apiserver)
A service meant to provide k8s External Metrics from CI providers API, which can be later used to configure autoscaling via Horizontal Pod Autoscalers.

## Currently supported providers
1. Fake-provider (used for testing)
2. Buildkite (WIP)
3. CircleCI (WIP)

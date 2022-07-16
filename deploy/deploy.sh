#!/bin/sh

set -e

readonly namespace=$1

if [ -z "$namespace" ]
then
    echo "usage: $0 <namespace>"
    exit 1
fi

kubectl_apply() {
    kubectl --namespace "$namespace" apply -f "$*"
}

kubectl_apply rbac-kube-system.yaml
kubectl_apply rbac.yaml
kubectl_apply service.yaml
kubectl_apply deployment.yaml
sed "s/##NAMESPACE##/$namespace/" < apiservice.yaml | kubectl_apply -

buildkite-deployment.yaml
flarebuild-deployment.yaml

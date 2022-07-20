#!/bin/sh

set -e

readonly namespace=$1

if [ -z "$namespace" ]
then
    echo "usage: $0 <namespace>"
    exit 1
fi

kubectl apply -f rbac-kube-system.yaml
kubectl_apply() {
    kubectl --namespace "$namespace" apply -f "$*"
}
sed "s/##NAMESPACE##/$namespace/" < rbac.yaml | tee /dev/stderr | kubectl_apply -
# kubectl_apply rbac.yaml
kubectl_apply service.yaml
kubectl_apply buildkite-deployment.yaml
sed "s/##NAMESPACE##/$namespace/" < apiservice.yaml | kubectl_apply -

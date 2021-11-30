DOCKER=docker
REGISTRY?=689494258501.dkr.ecr.us-east-1.amazonaws.com/elotl-dev
IMAGE?=buildscaler
BINARY_NAME=buildscaler

VERSION=$(shell git describe --dirty)

.PHONY: all
all: $(BINARY_NAME)

.PHONY: $(BINARY_NAME)
$(BINARY_NAME): pkg/generated/openapi/zz_generated.openapi.go main.go
	go build -o $(BINARY_NAME) main.go

pkg/generated/openapi/zz_generated.openapi.go: go.mod go.sum
	go install -mod=readonly k8s.io/kube-openapi/cmd/openapi-gen
	$(GOPATH)/bin/openapi-gen --logtostderr \
	    -i k8s.io/metrics/pkg/apis/custom_metrics,k8s.io/metrics/pkg/apis/custom_metrics/v1beta1,k8s.io/metrics/pkg/apis/custom_metrics/v1beta2,k8s.io/metrics/pkg/apis/external_metrics,k8s.io/metrics/pkg/apis/external_metrics/v1beta1,k8s.io/apimachinery/pkg/apis/meta/v1,k8s.io/apimachinery/pkg/api/resource,k8s.io/apimachinery/pkg/version,k8s.io/api/core/v1 \
	    -h ./openapi-gen-header.go.template \
	    -p ./pkg/generated/openapi \
	    -O zz_generated.openapi \
	    -o ./ \
	    -r /dev/null

.PHONY: bootstrap
bootstrap: $(GOPATH)/bin/goimports $(GOPATH)/bin/staticcheck

$(GOPATH)/bin/goimports:
	go get golang.org/x/tools/cmd/goimports

$(GOPATH)/bin/staticcheck:
	go install honnef.co/go/tools/cmd/staticcheck@2021.1.2

.PHONY: format
format:
	goimports -w $$(find . -type f -name '*.go' -not -path "./vendor/*")

.PHONY: check
check: format
	go vet ./...
	staticcheck ./...

.PHONY: verify
verify: verify-deps check

.PHONY: verify-deps
verify-deps:
	go mod verify
	go mod tidy
	@git diff --exit-code -- go.sum go.mod

.PHONY: test
test:
	go test -v -race -timeout=60s -cover  ./pkg/...

.PHONY: img
img:
	$(DOCKER) build -t $(REGISTRY)/$(IMAGE):$(VERSION) .

.PHONY: push-img
push-img: img
	docker push $(REGISTRY)/$(IMAGE):$(VERSION)

.PHONY: clean
	-rm -f $(BINARY_NAME)

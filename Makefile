DOCKER=docker
REGISTRY?=elotl
REPO?=buildscaler-dev
RELEASE_REPO?=buildscaler
BINARY_NAME=buildscaler

VERSION=$(shell git describe --tags --dirty)

.PHONY: all
all: $(BINARY_NAME)

.PHONY: $(BINARY_NAME)
$(BINARY_NAME): pkg/generated/openapi/zz_generated.openapi.go main.go
	go build -o $(BINARY_NAME) main.go

pkg/generated/openapi/zz_generated.openapi.go: go.mod go.sum
	go install -mod=readonly k8s.io/kube-openapi/cmd/openapi-gen@v0.0.0-20211115234752-e816edb12b65
	$(GOPATH)/bin/openapi-gen --logtostderr \
	    -i k8s.io/metrics/pkg/apis/custom_metrics,k8s.io/metrics/pkg/apis/custom_metrics/v1beta1,k8s.io/metrics/pkg/apis/custom_metrics/v1beta2,k8s.io/metrics/pkg/apis/external_metrics,k8s.io/metrics/pkg/apis/external_metrics/v1beta1,k8s.io/apimachinery/pkg/apis/meta/v1,k8s.io/apimachinery/pkg/api/resource,k8s.io/apimachinery/pkg/version,k8s.io/api/core/v1 \
	    -h ./openapi-gen-header.go.template \
	    -p ./pkg/generated/openapi \
	    -O zz_generated.openapi \
	    -o ./ \
	    -r /dev/null

$(GOPATH)/bin/goimports:
	go get golang.org/x/tools/cmd/goimports
	go install golang.org/x/tools/cmd/goimports

$(GOPATH)/bin/golangci-lint:
	go get github.com/golangci/golangci-lint/cmd/golangci-lint@v1.43.0
	go install github.com/golangci/golangci-lint/cmd/golangci-lint

format: $(GOPATH)/bin/goimports
	go run golang.org/x/tools/cmd/goimports \
        -w $$(find . -type f -name '*.go' -not -path "./vendor/*")

lint: $(GOPATH)/bin/golangci-lint
	go vet ./...
	go run github.com/golangci/golangci-lint/cmd/golangci-lint run ./...

check: format lint

.PHONY: format lint check

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
	$(DOCKER) build -t $(REGISTRY)/$(REPO):$(VERSION) .

.PHONY: push-img
push-img: img
	docker push $(REGISTRY)/$(REPO):$(VERSION)

release-img: img
	$(DOCKER) tag $(REGISTRY)/$(REPO):$(VERSION) $(REGISTRY)/$(RELEASE_REPO):$(VERSION)
	$(DOCKER) push $(REGISTRY)/$(RELEASE_REPO):$(VERSION)

.PHONY: clean
	-rm -f $(BINARY_NAME)

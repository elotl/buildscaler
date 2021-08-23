REGISTRY?=689494258501.dkr.ecr.us-east-1.amazonaws.com/elotl-dev
IMAGE?=ciplatforms-external-metrics
TEMP_DIR:=$(shell mktemp -d)
ARCH?=amd64
OUT_DIR?=./_output
BINARY_NAME=ciplatforms-external-metrics

VERSION?=latest

.PHONY: all
all: $(BINARY_NAME)

.PHONY: $(BINARY_NAME)
$(BINARY_NAME): pkg/generated/openapi/zz_generated.openapi.go
	CGO_ENABLED=0 GOOS=linux GOARCH=$(ARCH) go build -o $(OUT_DIR)/$(ARCH)/$(BINARY_NAME) main.go

pkg/generated/openapi/zz_generated.openapi.go: go.mod go.sum
	go install -mod=readonly k8s.io/kube-openapi/cmd/openapi-gen
	$(GOPATH)/bin/openapi-gen --logtostderr \
	    -i k8s.io/metrics/pkg/apis/custom_metrics,k8s.io/metrics/pkg/apis/custom_metrics/v1beta1,k8s.io/metrics/pkg/apis/custom_metrics/v1beta2,k8s.io/metrics/pkg/apis/external_metrics,k8s.io/metrics/pkg/apis/external_metrics/v1beta1,k8s.io/apimachinery/pkg/apis/meta/v1,k8s.io/apimachinery/pkg/api/resource,k8s.io/apimachinery/pkg/version,k8s.io/api/core/v1 \
	    -h ./hack/boilerplate.go.txt \
	    -p ./pkg/generated/openapi \
	    -O zz_generated.openapi \
	    -o ./ \
	    -r /dev/null

.PHONY: gofmt
gofmt:
	./hack/gofmt-all.sh

.PHONY: verify
verify: verify-deps verify-gofmt

.PHONY: verify-deps
verify-deps:
	go mod verify
	go mod tidy
	@git diff --exit-code -- go.sum go.mod

.PHONY: verify-gofmt
verify-gofmt:
	./hack/gofmt-all.sh -v

.PHONY: test
test: $(BINARY_NAME)
	CGO_ENABLED=0 go test ./pkg/...

.PHONY: img
img: $(BINARY_NAME)
	cp deploy/Dockerfile $(TEMP_DIR)
	cp $(OUT_DIR)/$(ARCH)/$(BINARY_NAME) $(TEMP_DIR)/adapter
	cd $(TEMP_DIR)
	docker build -t $(REGISTRY)/$(IMAGE)-$(ARCH):$(VERSION) $(TEMP_DIR)
	rm -rf $(TEMP_DIR)

.PHONY: push-img
push-img: img
	docker push $(REGISTRY)/$(IMAGE)-$(ARCH):$(VERSION)


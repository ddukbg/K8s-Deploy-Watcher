# Image URL to use all building/pushing image targets
IMG ?= controller:latest

# ENVTEST_K8S_VERSION refers to the version of kubebuilder assets to be downloaded by envtest binary.
ENVTEST_K8S_VERSION = 1.24

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

# 도커 이미지 태그 설정
VERSION ?= $(shell git describe --tags --always --dirty)
IMG_TAG ?= $(VERSION)

.PHONY: all
all: build test

.PHONY: fmt
fmt:
	go fmt ./...

.PHONY: vet
vet:
	go vet ./...

.PHONY: test
test: fmt vet
	go test ./... -coverprofile cover.out

.PHONY: build
build:
	go build -o bin/manager main.go

.PHONY: docker-build
docker-build:
	docker build -t ${IMG}:${IMG_TAG} .

.PHONY: docker-push
docker-push:
	docker push ${IMG}:${IMG_TAG}

.PHONY: install
install:
	kubectl apply -f config/crd/deployment_tracker.yaml

.PHONY: uninstall
uninstall:
	kubectl delete -f config/crd/deployment_tracker.yaml

.PHONY: deploy
deploy: docker-build docker-push install
	kubectl apply -f config/rbac/
	cat config/manager/manager.yaml | sed 's|IMAGE_TAG|${IMG_TAG}|g' | kubectl apply -f -

.PHONY: undeploy
undeploy:
	kubectl delete -f config/manager/manager.yaml
	kubectl delete -f config/rbac/
	make uninstall

# 로컬 개발용 타겟 추가
.PHONY: run
run:
	go run ./main.go

# Generate code
.PHONY: generate
generate:
	controller-gen object:headerFile="hack/boilerplate.go.txt" paths="./..."

# Generate manifests
.PHONY: manifests
manifests:
	controller-gen crd paths="./..." output:crd:artifacts:config=config/crd/bases

# 통합 테스트
.PHONY: integration-test
integration-test:
	go test ./... -tags=integration

# 코드 커버리지 리포트
.PHONY: coverage
coverage:
	go tool cover -html=cover.out
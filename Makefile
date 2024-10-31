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

# Tools 설정
CONTROLLER_GEN = $(GOBIN)/controller-gen
ENVTEST = $(GOBIN)/setup-envtest

.PHONY: all
all: generate fmt vet build test

.PHONY: fmt
fmt:
	go fmt ./...

.PHONY: vet
vet:
	go vet ./...

.PHONY: test
test: generate fmt vet
	go test ./... -coverprofile cover.out

.PHONY: build
build: generate
	go build -o bin/manager main.go

# controller-gen 설치
.PHONY: controller-gen
controller-gen:
	GOBIN=$(GOBIN) go install sigs.k8s.io/controller-tools/cmd/controller-gen@v0.14.0

# Generate code and manifests
.PHONY: generate
generate: controller-gen
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./api/..."
	$(CONTROLLER_GEN) rbac:roleName=manager-role crd paths="./api/..." output:crd:artifacts:config=config/crd/bases

.PHONY: manifests
manifests: controller-gen
	$(CONTROLLER_GEN) crd paths="./..." output:crd:artifacts:config=config/crd/bases

# Docker 관련 타겟
.PHONY: docker-build
docker-build:
	docker build -t $(IMG) .

.PHONY: docker-push
docker-push:
	docker push $(IMG)

# 버전 태그 추가
.PHONY: docker-tag
docker-tag:
	docker tag $(IMG) $(IMG):$(VERSION)

# 모든 Docker 작업 실행
.PHONY: docker-all
docker-all: docker-build docker-tag docker-push

.PHONY: install
install:
	kubectl apply -f config/crd/bases/ddukbg.k8s_deploymenttrackers.yaml

.PHONY: uninstall
uninstall:
	kubectl delete -f config/crd/bases/ddukbg.k8s_deploymenttrackers.yaml

.PHONY: deploy
deploy: manifests docker-build docker-push install
	kubectl apply -f config/rbac/
	cat config/manager/manager.yaml | sed 's|IMAGE_TAG|${IMG_TAG}|g' | kubectl apply -f -

.PHONY: undeploy
undeploy:
	kubectl delete -f config/manager/manager.yaml
	kubectl delete -f config/rbac/
	make uninstall

# 로컬 개발용 타겟 추가
.PHONY: run
run: generate
	go run ./main.go

# 통합 테스트
.PHONY: integration-test
integration-test:
	go test ./... -tags=integration

# 코드 커버리지 리포트
.PHONY: coverage
coverage:
	go tool cover -html=cover.out

# 로컬 테스트
.PHONY: local-test
local-test: generate fmt vet
	go mod verify
	go mod tidy
	go test ./... -v
	go build -v .

# GitHub Actions workflow 로컬 테스트용
.PHONY: test-workflow
test-workflow:
	act -j test
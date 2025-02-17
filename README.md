***개인적인 용도로 사용하기 위해 계속해서 개발 중입니다.***
# K8s-Deploy-Watcher

![Go Version](https://img.shields.io/badge/go-v1.22.1-blue.svg)
![Kubernetes](https://img.shields.io/badge/kubernetes-%3E%3D1.21-blue.svg)
![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)

## 목차

- [주요 기능](#-주요-기능)
- [시스템 요구사항](#-시스템-요구사항)
- [배포 가이드](#-배포-가이드)
  - [사전 요구사항](#1-사전-요구사항)
  - [RBAC 및 서비스 어카운트 설정](#2-rbac-및-서비스-어카운트-설정)
  - [이미지 빌드 및 푸시](#3-이미지-빌드-및-푸시)
  - [CRD 및 Controller 배포](#4-crd-및-controller-배포)
  - [설치 확인](#5-설치-확인)
  - [문제 해결](#6-문제-해결)
  - [제거](#7-제거)
- [사용 방법](#-사용-방법)
  - [단일 리소스 모니터링](#1-resourcetracker-생성---단일-리소스-모니터링)
  - [네임스페이스 전체 모니터링](#2-resourcetracker-생성---네임스페이스-전체-모니터링)
- [상태 확인](#-상태-확인)
- [모니터링 동작 방식](#-모니터링-동작-방식)
- [개발 환경 설정](#-개발-환경-설정)
- [라이선스](#-라이선스)

K8s-Deploy-Watcher는 Kubernetes 리소스의 실시간 상태를 모니터링하고 
상태 변경을 Slack으로 알려주는 Custom Controller입니다.

> 🔨 현재 개발 상태:
> - Custom Resource Definition (CRD) 구현
> - Custom Controller 구현
> - 향후 완전한 Operator 패턴으로 발전 예정

## 🌟 주요 기능

- **다양한 리소스 모니터링**
  - Deployment
  - StatefulSet
  - Pod

- **모니터링 범위**
  - 단일 리소스 모니터링
  - 네임스페이스 전체 리소스 모니터링 (신규)

- **알림 기능**
  - Slack 웹훅 지원
  - 리소스별 맞춤 알림 메시지
  - 상태 변경 실시간 알림

## 💻 시스템 요구사항

- Kubernetes >= 1.21
- Go >= 1.20

## 📦 배포 가이드

### 1. 사전 요구사항
- kubectl이 설치되어 있고 클러스터에 접근 가능
- 클러스터 관리자 권한
- make 명령어 사용 가능

### 2. RBAC 및 서비스 어카운트 설정

```bash
# namespace 생성
kubectl create namespace k8s-deploy-watcher-system

# RBAC 설정 적용
kubectl apply -f config/rbac/role.yaml
kubectl apply -f config/rbac/role_binding.yaml
kubectl apply -f config/rbac/service_account.yaml
```

필요한 RBAC 권한:
```yaml
# config/rbac/role.yaml 예시
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: k8s-deploy-watcher-role
rules:
  - apiGroups: ["ddukbg.k8s"]
    resources: ["resourcetrackers"]
    verbs: ["get", "list", "watch", "create", "update", "patch", "delete"]
  - apiGroups: [""]
    resources: ["pods"]
    verbs: ["get", "list", "watch"]
  - apiGroups: ["apps"]
    resources: ["deployments", "statefulsets"]
    verbs: ["get", "list", "watch"]
  - apiGroups: [""]
    resources: ["events"]
    verbs: ["create", "patch"]
```

### 3. 이미지 빌드 및 푸시
```bash
# 도커 이미지 빌드
make docker-build IMG=your-registry/k8s-deploy-watcher:tag

# 도커 이미지 푸시
make docker-push IMG=your-registry/k8s-deploy-watcher:tag
```

### 4. CRD 및 Controller 배포

```bash
# CRD 설치
make install

# Controller 배포
make deploy IMG=your-registry/k8s-deploy-watcher:tag

# 배포 확인
kubectl get deployment -n k8s-deploy-watcher-system
kubectl get serviceaccount -n k8s-deploy-watcher-system
kubectl get clusterrole k8s-deploy-watcher-role
kubectl get clusterrolebinding k8s-deploy-watcher-rolebinding
```

### 5. 설치 확인
```bash
# CRD 확인
kubectl get crd resourcetrackers.ddukbg.k8s

# Controller Pod 확인
kubectl get pods -n k8s-deploy-watcher-system

# 로그 확인
kubectl logs -n k8s-deploy-watcher-system deployment/k8s-deploy-watcher-controller-manager -c manager
```

### 6. 문제 해결

일반적인 문제 해결 단계:
1. Pod 상태 확인
```bash
kubectl get pods -n k8s-deploy-watcher-system
kubectl describe pod -n k8s-deploy-watcher-system <pod-name>
```

2. 로그 확인
```bash
kubectl logs -n k8s-deploy-watcher-system deployment/k8s-deploy-watcher-controller-manager -c manager --tail=100
```

3. RBAC 권한 확인
```bash
kubectl auth can-i get pods --as=system:serviceaccount:k8s-deploy-watcher-system:k8s-deploy-watcher-controller-manager
```

### 7. 제거
```bash
# Controller 제거
make undeploy

# CRD 제거
make uninstall

# RBAC 설정 제거
kubectl delete -f config/rbac/role.yaml
kubectl delete -f config/rbac/role_binding.yaml
kubectl delete -f config/rbac/service_account.yaml

# Namespace 제거
kubectl delete namespace k8s-deploy-watcher-system
```

## 🚀 사용 방법

### 1. ResourceTracker 생성 - 단일 리소스 모니터링

```yaml
apiVersion: ddukbg.k8s/v1alpha1
kind: ResourceTracker
metadata:
  name: nginx-tracker
  namespace: default
spec:
  target:
    kind: Deployment # Deployment, StatefulSet, Pod
    name: nginx      # 특정 리소스 이름
    namespace: default
  notify:
    slack: "https://hooks.slack.com/services/..."
```

### 2. ResourceTracker 생성 - 네임스페이스 전체 모니터링

```yaml
apiVersion: ddukbg.k8s/v1alpha1
kind: ResourceTracker
metadata:
  name: namespace-pods-tracker
  namespace: monitoring
spec:
  target:
    kind: Pod          # Deployment, StatefulSet, Pod
    namespace: default # 모니터링할 네임스페이스
  notify:
    slack: "https://hooks.slack.com/services/..."
```

## 🔍 상태 확인

```bash
# ResourceTracker 상태 확인
kubectl get resourcetracker

# 특정 ResourceTracker 상세 정보
kubectl describe resourcetracker <name>
```


## 📊 모니터링 동작 방식

1. **리소스 감지**
   - 지정된 리소스의 상태 변경 감지
   - 네임스페이스 전체 모니터링 시 해당 타입의 모든 리소스 감지

2. **상태 체크**
   - Deployment: ReadyReplicas, UpdatedReplicas, AvailableReplicas 확인
   - StatefulSet: ReadyReplicas, UpdatedReplicas 확인
   - Pod: Running 상태 확인

3. **알림 발송**
   - 리소스가 Ready 상태가 되면 Slack 알림 발송
   - 리소스별 맞춤 메시지 포맷 사용

## 🔧 개발 환경 설정
```bash
# 의존성 설치
go mod download
# 테스트 실행
make test
# 로컬 실행
make run
```


## 📜 라이선스

이 프로젝트는 MIT 라이선스로 제공됩니다. 자세한 내용은 [LICENSE](LICENSE) 파일을 참조하세요.

![Go Version](https://img.shields.io/github/go-mod/go-version/golang/go)
![Kubernetes](https://img.shields.io/badge/kubernetes-operator-blue.svg)
![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)


***개인적인 용도로 사용하기 위해 계속해서 개발 중입니다.***
## 📋 K8s-Deploy-Watcher

**K8s-Deploy-Watcher**는 CI/CD 파이프라인과 함께 동작하여 **실제 배포** 상태를 추적하고, 배포 작업이 성공적으로 완료되었을 때 또는 오류가 발생했을 때 알림을 제공합니다.

### 🚀 **왜 필요한가?**

기존의 배포 상태 모니터링은 단순히 `Deployment` 리소스의 상태를 확인하지만, 이 오퍼레이터는 **CI/CD 파이프라인에서 트리거된 배포 작업을 정확하게 추적**하는 데 중점을 둡니다. 이를 통해 사용자는 배포가 실제로 **이미지 커밋이 변경**되었는지, **스케일링 작업**과는 무관하게 **정확한 배포 상태**만을 추적할 수 있습니다.

### **주요 기능**

- **CI/CD 배포 추적**: 새로운 이미지가 커밋되거나 배포가 시작될 때 정확히 그 배포 상태를 추적합니다.
- **Slack 및 Email 알림**: 배포 성공, 실패 등의 이벤트에 대해 실시간으로 알림을 받을 수 있습니다.
- **유연한 배포 추적**: 특정 네임스페이스의 특정 애플리케이션만 추적하거나, 전체 클러스터의 배포 상태를 추적할 수 있습니다.

---

## 📦 **구조**

프로젝트 구조
```plaintext
k8s-deployment-tracker/
├── config/
│   ├── crd/                   # Custom Resource Definition YAML 파일
│   │   └── deployment_tracker.yaml  # CRD 정의
│   ├── rbac/                  # Operator의 권한 설정 (RBAC)
│   └── samples/               # CR 예제 파일들
│       ├── deployment_tracker_my_app.yaml   # 특정 Deployment 추적 CR 예시
│       └── deployment_tracker_wildcard.yaml # 범용 추적 CR 예시
├── controllers/
│   ├── deployment_tracker.go  # Operator 핵심 로직
├── api/
│   ├── v1alpha1/              # CRD 스키마 정의
│   └── groupversion_info.go
├── main.go                    # Operator 진입점
├── Dockerfile                 # 도커 이미지 설정
└── Makefile                   # 빌드, 테스트 자동화
```

배포가 어떻게 이루어지고 K8s-Deploy-Watcher가 이를 어떻게 추적하는지 간단한 텍스트 기반 구조도로 설명할 수 있습니다.
```plaintext
┌─────────────────────────┐
│                         │
│    CI/CD 파이프라인        │
│  (이미지 빌드 및 커밋)       │
│                         │
└──────────────┬──────────┘
               │
               │
               ▼
   ┌────────────────────────┐    배포 시작!
   │  Kubernetes Cluster     │─────────────────────┐
   │ (Deployment 생성 및 관리)  │                     │
   └───────────┬─────────────┘                     ▼
               │
               │                        ┌────────────────────────┐
               ▼                        │                        │
      ┌──────────────────────────┐      │  K8s-Deploy-Watcher    │
      │   실제 배포 상태 추적         │      │                        │
      │   (이미지 커밋 변경 확인)     │      │  (배포 완료 시 알림)       │
      └──────────────────────────┘      └─────────────┬──────────┘
               │                                      │
               ▼                                      ▼
   ┌────────────────────────┐            ┌────────────────────────┐
   │   배포 성공 또는 실패       │            │   Slack 또는 Email 알림   │
   │    (Running/Succeeded) │            │                        │
   └────────────────────────┘            └────────────────────────┘
```

- **CI/CD 파이프라인**: 새로운 이미지가 커밋되고 배포가 시작됩니다.
- **Kubernetes Cluster**: 클러스터에서 실제로 Deployment가 이루어집니다.
- **K8s-Deploy-Watcher**: 이 Operator는 배포 상태를 추적하고, 배포 완료 시 알림을 발송합니다.
- **Slack 또는 Email 알림**: 배포가 완료되었거나 실패했을 때 실시간 알림을 제공합니다.

---

## 📋 **설치 및 사용법**

### **1. 설치 방법**

1. 이 저장소를 클론합니다:

   ```bash
   git clone https://github.com/ddukbg/k8s-deploy-watcher.git
   cd k8s-deploy-watcher
   ```

2. 오퍼레이터를 빌드하고 Kubernetes 클러스터에 배포합니다:

   ```bash
   # Docker 이미지 빌드
   make build

   # CRD 및 오퍼레이터 배포
   make deploy
   ```

3. Custom Resource (CR)를 적용하여 배포 상태를 추적합니다:

   ```bash
   kubectl apply -f config/samples/deployment_tracker_my_app.yaml
   ```

### **2. 사용 방법**

#### 특정 배포 추적

```yaml
apiVersion: ddukbg/v1alpha1
kind: DeploymentTracker
metadata:
  name: my-app-deployment-tracker
spec:
  deploymentName: my-app
  namespace: default
  notify:
    slack: "https://hooks.slack.com/services/..."
    email: "alert@example.com"
```

#### 모든 배포 추적

```yaml
apiVersion: ddukbg/v1alpha1
kind: DeploymentTracker
metadata:
  name: all-deployments-tracker
spec:
  notify:
    slack: "https://hooks.slack.com/services/..."
    email: "alert@example.com"
```

---

## ⚙️ **개발 및 기여**

이 오퍼레이터는 로컬 환경에서 개발할 수 있으며, 기여도 환영합니다:

```bash
# 오퍼레이터 로컬 실행
make run
```

## 🙌 **기여**

기여는 언제나 환영합니다! 버그를 발견하거나 기능 제안이 있다면 이슈를 생성하거나 풀 리퀘스트를 제출해 주세요.

## 📝 **라이선스**

이 프로젝트는 MIT 라이선스 하에 제공됩니다. 자세한 내용은 [LICENSE](LICENSE) 파일을 참고하세요.

## 💡 **감사의 말**

- [Kubernetes](https://kubernetes.io/)
- [Go](https://golang.org/)
- [Controller Runtime](https://github.com/kubernetes-sigs/controller-runtime)


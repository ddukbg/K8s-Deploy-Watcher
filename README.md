***개인적인 용도로 사용하기 위해 계속해서 개발 중입니다.***
# K8s-Deploy-Watcher

![Go Version](https://img.shields.io/badge/go-v1.20-blue.svg)
![Kubernetes](https://img.shields.io/badge/kubernetes-%3E%3D1.21-blue.svg)
![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)
![Docker Pulls](https://img.shields.io/docker/pulls/ddukbg/k8s-deploy-watcher)
![Tests](https://github.com/ddukbg/k8s-deploy-watcher/workflows/Test/badge.svg)

K8s-Deploy-Watcher는 Kubernetes Deployment의 실시간 상태를 모니터링하고 배포 결과를 Slack 또는 이메일로 알려주는 Kubernetes Operator입니다.

## 🌟 주요 기능

- **실시간 배포 모니터링**
  - Deployment 상태 실시간 추적
  - 배포 성공/실패 즉시 감지
  - 사용자 정의 알림 조건 설정

- **다양한 알림 채널**
  - Slack 웹훅 지원
  - 이메일 알림 지원
  - 알림 재시도 메커니즘

- **상세한 상태 정보**
  - 상세한 배포 상태 정보 제공
  - 실시간 메트릭스 수집
  - Kubernetes 이벤트와 통합

## 💻 시스템 요구사항

- Kubernetes >= 1.21
- Go >= 1.20 (개발 시)
- kubectl CLI
- Helm (선택사항)

## 🚀 설치 방법

### Helm을 사용한 설치

```bash
# 프로젝트 클론
git clone https://github.com/ddukbg/k8s-deploy-watcher.git
cd k8s-deploy-watcher

# Helm 차트 설치
helm install deploy-watcher ./charts/k8s-deploy-watcher \
  --namespace deploy-watcher \
  --create-namespace

# Operator 설치
helm install deploy-watcher ddukbg/k8s-deploy-watcher \
  --namespace deploy-watcher \
  --create-namespace
```

### 수동 설치

```bash
# 1. CRD 및 RBAC 설정 적용
kubectl apply -f config/crd/deployment_tracker.yaml
kubectl apply -f config/rbac/role.yaml
kubectl apply -f config/rbac/role_binding.yaml

# 2. Operator Deployment 생성 및 적용
cat <<EOF > config/manager/manager.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: k8s-deploy-watcher
  namespace: default
spec:
  replicas: 1
  selector:
    matchLabels:
      app: k8s-deploy-watcher
  template:
    metadata:
      labels:
        app: k8s-deploy-watcher
    spec:
      serviceAccountName: deployment-tracker
      containers:
      - name: manager
        image: ${AWS_ACCOUNT_ID}.dkr.ecr.${AWS_REGION}.amazonaws.com/k8s-deploy-watcher:latest
        ports:
        - containerPort: 8080
          name: metrics
        - containerPort: 8081
          name: health
        livenessProbe:
          httpGet:
            path: /healthz
            port: 8081
        readinessProbe:
          httpGet:
            path: /readyz
            port: 8081
        resources:
          limits:
            cpu: 500m
            memory: 256Mi
          requests:
            cpu: 200m
            memory: 128Mi
EOF

kubectl apply -f config/manager/manager.yaml

# 3. Operator Pod 실행 상태 확인
kubectl get pods -l app=k8s-deploy-watcher
kubectl logs -l app=k8s-deploy-watcher

# 4. 테스트용 Deployment 생성
cat <<EOF > nginx-deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx
spec:
  replicas: 3
  selector:
    matchLabels:
      app: nginx
  template:
    metadata:
      labels:
        app: nginx
    spec:
      containers:
      - name: nginx
        image: nginx:latest
        ports:
        - containerPort: 80
EOF

kubectl apply -f nginx-deployment.yaml

# 5. DeploymentTracker CR 생성
cat <<EOF > tracker-example.yaml
apiVersion: ddukbg.k8s/v1alpha1
kind: DeploymentTracker
metadata:
  name: nginx-tracker
spec:
  deploymentName: nginx
  namespace: default
  notify:
    slack: "https://hooks.slack.com/services/YOUR-WEBHOOK-URL"
    retryCount: 3
    alertOnFail: true
EOF

kubectl apply -f tracker-example.yaml

# 6. DeploymentTracker 상태 확인
kubectl get deploymenttracker
kubectl describe deploymenttracker nginx-tracker

# 7. 배포 변경으로 테스트
kubectl set image deployment/nginx nginx=nginx:1.25.0

# 8. 로그 및 Slack 알림 확인
kubectl logs -l app=k8s-deploy-watcher
```

## 📋 사용 방법

### 1. DeploymentTracker 리소스 생성

```yaml
# Deployment 단일 대상으로 지정
apiVersion: ddukbg.k8s/v1alpha1
kind: DeploymentTracker
metadata:
  name: my-app-tracker
spec:
  deploymentName: my-app
  namespace: default
  notify:
    slack: "https://hooks.slack.com/services/..."
    email: "alert@example.com"
    retryCount: 3
    alertOnFail: true
```

```yaml
# All 모든 배포 대상으로 지정(미구현)
apiVersion: ddukbg/v1alpha1
kind: DeploymentTracker
metadata:
  name: all-deployments-tracker
spec:
  notify:
    slack: "https://hooks.slack.com/services/..."
    email: "alert@example.com"
```

### 2. 상태 확인

```bash
# Tracker 상태 확인
kubectl get deploymenttracker

# 상세 정보 확인
kubectl describe deploymenttracker my-app-tracker
```

### 3. 알림 설정

#### Slack 알림 설정
1. Slack 앱 설정에서 Incoming Webhook URL 생성
2. DeploymentTracker 리소스의 `spec.notify.slack`에 URL 설정

#### 이메일 알림 설정
1. SMTP 서버 정보를 Secret으로 생성
```bash
kubectl create secret generic smtp-config \
  --from-literal=host=smtp.gmail.com \
  --from-literal=port=587 \
  --from-literal=username=your-email@gmail.com \
  --from-literal=password=your-app-password
```
2. DeploymentTracker 리소스의 `spec.notify.email`에 수신자 이메일 설정

## 🔧 개발 환경 설정

### 로컬 개발 환경 설정

```bash
# 의존성 설치
go mod download

# 코드 생성
make generate

# CRD 매니페스트 생성
make manifests

# 로컬에서 실행
make run
```

### 테스트 실행

```bash
# 단위 테스트
make test

# 통합 테스트
make integration-test

# 커버리지 리포트 생성
make coverage
```

## 📊 모니터링 및 메트릭스

Operator는 다음 엔드포인트를 제공합니다:
- Health check: `:8081/healthz`
- Metrics: `:8080/metrics`
- Ready check: `:8081/readyz`

Prometheus와 통합하여 메트릭스를 수집할 수 있습니다:
```yaml
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: deploy-watcher-metrics
spec:
  endpoints:
  - port: metrics
  selector:
    matchLabels:
      app: deploy-watcher
```

## 🔍 트러블슈팅

### 일반적인 문제 해결

1. Operator Pod가 시작되지 않는 경우
```bash
kubectl describe pod -n deploy-watcher
kubectl logs -n deploy-watcher <pod-name>
```

2. 알림이 발송되지 않는 경우
- Slack Webhook URL 확인
- SMTP 설정 확인
- 네트워크 정책 확인

### 로그 확인

```bash
# Operator 로그 확인
kubectl logs -f deployment/deploy-watcher-controller-manager -n deploy-watcher

# 이벤트 확인
kubectl get events --field-selector involvedObject.kind=DeploymentTracker
```

## 🤝 기여하기

1. Fork 생성
2. Feature 브랜치 생성 (`git checkout -b feature/amazing-feature`)
3. 변경사항 커밋 (`git commit -m 'Add amazing feature'`)
4. 브랜치에 Push (`git push origin feature/amazing-feature`)
5. Pull Request 생성

### 코드 스타일

- [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments) 준수
- 모든 코드는 `go fmt`와 `go vet` 통과 필요
- 단위 테스트 필수

## 📜 라이선스

이 프로젝트는 MIT 라이선스로 제공됩니다. 자세한 내용은 [LICENSE](LICENSE) 파일을 참조하세요.

## 🙏 감사의 글

- [Kubernetes](https://kubernetes.io/)
- [Operator SDK](https://sdk.operatorframework.io/)
- [Controller Runtime](https://github.com/kubernetes-sigs/controller-runtime)
- [Kubebuilder](https://kubebuilder.io/)

## 📞 문의하기

- Issue 생성: [GitHub Issues](https://github.com/ddukbg/k8s-deploy-watcher/issues)
- 이메일: wowrebong@gmail.com
- Slack: [Kubernetes Slack #deploy-watcher](https://kubernetes.slack.com/messages/deploy-watcher)

---
⭐️ 이 프로젝트가 유용하다면 스타를 눌러주세요!
# Makefile

# 빌드
build:
	docker build -t k8s-deploy-watcher .

# 유닛 테스트 실행
test:
	go test ./controllers/... -v

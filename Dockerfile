# Build stage
# Go 버전 업데이트
FROM golang:1.22.1 as builder

WORKDIR /workspace
# 캐시 활용을 위해 go.mod와 go.sum을 먼저 복사
COPY go.mod go.sum ./
RUN go mod download

# 소스 코드 복사
COPY . .

# 빌드
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 GO111MODULE=on go build -a -o manager main.go

# Runtime stage
FROM gcr.io/distroless/static:nonroot
WORKDIR /
COPY --from=builder /workspace/manager .

# 보안을 위한 사용자 설정
USER 65532:65532

# 헬스체크 포트 노출
EXPOSE 8080 8081

ENTRYPOINT ["/manager"]
# Makefile for API Bridge

.PHONY: help run build test clean install-tools

# 기본 변수
APP_NAME=api-bridge
BINARY_NAME=api-bridge.exe
CMD_DIR=./cmd/api-bridge
BUILD_DIR=./bin
GO=go

help: ## 도움말 표시
	@echo "Available commands:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2}'

run: ## 개발 모드로 실행 (핫 리로드)
	air

run-direct: ## Air 없이 직접 실행
	$(GO) run $(CMD_DIR)/main.go

build: ## 프로덕션 빌드
	@echo "Building $(APP_NAME)..."
	@mkdir -p $(BUILD_DIR)
	$(GO) build -ldflags="-s -w" -o $(BUILD_DIR)/$(BINARY_NAME) $(CMD_DIR)/main.go
	@echo "Build complete: $(BUILD_DIR)/$(BINARY_NAME)"

test: ## 테스트 실행
	$(GO) test -v -race -coverprofile=coverage.out ./...

test-coverage: test ## 테스트 커버리지 확인
	$(GO) tool cover -html=coverage.out

test-db: ## OracleDB 연결 테스트
	$(GO) run ./cmd/test-db/main.go

test-redis: ## Redis 연결 테스트
	$(GO) test -v ./test/redis_test.go

test-all-db: ## 모든 데이터베이스 연결 테스트
	$(GO) run ./cmd/test-all/main.go

test-unit: ## 단위 테스트만 실행 (DB 연결 제외)
	$(GO) test -v -race -coverprofile=coverage.out ./pkg/... ./internal/...

test-integration: ## 통합 테스트 실행 (DB 연결 포함)
	$(GO) test -v -race -coverprofile=coverage.out ./test/...

lint: ## 린트 실행
	golangci-lint run ./...

fmt: ## 코드 포맷팅
	gofmt -s -w .
	goimports -w .

tidy: ## 의존성 정리
	$(GO) mod tidy
	$(GO) mod verify

clean: ## 빌드 결과물 삭제
	@echo "Cleaning..."
	@rm -rf $(BUILD_DIR) tmp coverage.out
	@echo "Clean complete"

install-tools: ## 개발 도구 설치
	$(GO) install github.com/cosmtrek/air@latest
	$(GO) install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	$(GO) install golang.org/x/tools/cmd/goimports@latest

deps: ## 의존성 다운로드
	$(GO) mod download

docker-build: ## Docker 이미지 빌드
	docker build -t $(APP_NAME):latest .

docker-run: ## Docker 컨테이너 실행
	docker run -p 10019:10019 $(APP_NAME):latest

.DEFAULT_GOAL := help


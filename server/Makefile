APP_NAME := chengzi-agent-code-server
CMD_PATH := ./cmd/server
BUILD_DIR := build
CONFIG ?= etc/ade-dev.yaml

.PHONY: run dev build clean tidy fmt

run:
	go run $(CMD_PATH) -config $(CONFIG)

# 生产配置示例：make run CONFIG=etc/ade.yaml
# 热重启开发（依赖 air：go install github.com/air-verse/air@latest）
dev:
	@command -v air >/dev/null 2>&1 || { echo "air 未安装，执行: go install github.com/air-verse/air@latest"; exit 1; }
	air

tidy:
	go mod tidy

fmt:
	go fmt ./...

build:
	mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o $(BUILD_DIR)/$(APP_NAME)-linux-amd64 $(CMD_PATH)
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -trimpath -ldflags="-s -w" -o $(BUILD_DIR)/$(APP_NAME)-linux-arm64 $(CMD_PATH)
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o $(BUILD_DIR)/$(APP_NAME)-darwin-amd64 $(CMD_PATH)
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -trimpath -ldflags="-s -w" -o $(BUILD_DIR)/$(APP_NAME)-darwin-arm64 $(CMD_PATH)
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o $(BUILD_DIR)/$(APP_NAME)-windows-amd64.exe $(CMD_PATH)
	CGO_ENABLED=0 GOOS=windows GOARCH=arm64 go build -trimpath -ldflags="-s -w" -o $(BUILD_DIR)/$(APP_NAME)-windows-arm64.exe $(CMD_PATH)

clean:
	rm -rf $(BUILD_DIR)

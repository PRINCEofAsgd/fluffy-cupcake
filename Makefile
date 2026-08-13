.PHONY: dev test build docker-build docker-up docker-down

APP_VERSION ?= V0.0.2_20260814
TARGET_PLATFORM ?= linux/amd64
IMAGE ?= fluffy-cupcake:$(APP_VERSION)

# 本地开发时监听全部网口的 4819 端口。
dev:
	APP_ADDR=0.0.0.0:4819 APP_MODE=debug go run ./cmd/server

test:
	go test ./...

build:
	APP_VERSION=$(APP_VERSION) TARGET_PLATFORM=$(TARGET_PLATFORM) IMAGE=$(IMAGE) ./scripts/build.sh

# 保留原命令名作为兼容别名，实际同样只构建 Linux 镜像。
docker-build:
	$(MAKE) build

docker-up:
	docker compose up -d --build

docker-down:
	docker compose down

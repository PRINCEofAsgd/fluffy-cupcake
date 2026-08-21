.PHONY: dev test build db-up migrate-up migrate-down create-user create-qr-login docker-build docker-up docker-down

APP_VERSION ?= V1.0.0_20260822
TARGET_PLATFORM ?= linux/amd64
IMAGE ?= fluffy-cupcake:$(APP_VERSION)

# 本地开发时监听全部网口的 4819 端口。
dev:
	./scripts/dev.sh

test:
	go test ./...

build:
	APP_VERSION=$(APP_VERSION) TARGET_PLATFORM=$(TARGET_PLATFORM) IMAGE=$(IMAGE) ./scripts/build.sh

# 启动本地 MySQL；凭据从不会提交的 .env 读取。
db-up:
	docker compose up -d mysql

migrate-up:
	docker compose --profile tools run --rm migrate up

migrate-down:
	docker compose --profile tools run --rm migrate down 1

create-user:
	@test -n "$(USERNAME)" || (echo "用法: make create-user USERNAME=用户名" >&2; exit 1)
	@set -a; . ./.env; set +a; go run ./cmd/create-user --username "$(USERNAME)"

# 为二维码卡片建立文本到用户的映射；二维码文本较长时可使用 --qr-text-stdin。
create-qr-login:
	@test -n "$(USERNAME)" || (echo "用法: make create-qr-login USERNAME=用户名" >&2; exit 1)
	@set -a; . ./.env; set +a; go run ./cmd/create-qr-login --username "$(USERNAME)" $(QR_TEXT_FLAGS)

# 保留原命令名作为兼容别名，实际同样只构建 Linux 镜像。
docker-build:
	$(MAKE) build

docker-up:
	docker compose up -d --build

docker-down:
	docker compose down

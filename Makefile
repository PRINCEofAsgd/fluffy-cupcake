.PHONY: dev test build homepage-build db-up migrate-up migrate-down create-user create-qr-login db-preflight db-verify docker-build docker-up docker-down

APP_VERSION ?= V1.1.0_20261115
TARGET_PLATFORM ?= linux/amd64
IMAGE ?= fluffy-cupcake:$(APP_VERSION)

# 本地开发时监听全部网口的 4819 端口。
HOMEPAGE_DEPS_STAMP := web/homepage/node_modules/.package-lock.json

$(HOMEPAGE_DEPS_STAMP): web/homepage/package.json web/homepage/package-lock.json
	npm --prefix web/homepage ci

# 构建脱敏 Homepage，并把 Vite 产物交给 Go embed 打包。
homepage-build: $(HOMEPAGE_DEPS_STAMP)
	npm --prefix web/homepage run build

dev: homepage-build
	./scripts/dev.sh

test: homepage-build
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

# 对指定 OLD/NEW 数据库执行只读预检；连接信息通过环境变量提供，密码由终端隐藏读取。
db-preflight:
	@test -n "$(SIDE)" || (echo "用法: make db-preflight SIDE=OLD 或 SIDE=NEW" >&2; exit 1)
	./scripts/db-migration/verify.sh preflight "$(SIDE)"

# 对 OLD/NEW 生成只读快照并比较表、行数、Migration、索引、外键和 UTC 时间边界。
db-verify:
	./scripts/db-migration/verify.sh verify

# 保留原命令名作为兼容别名，实际同样只构建 Linux 镜像。
docker-build:
	$(MAKE) build

docker-up:
	docker compose up -d --build

docker-down:
	docker compose down

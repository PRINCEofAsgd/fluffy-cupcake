# 构建阶段固定 Go 工具链，并利用缓存加速依赖下载和重复构建。
FROM --platform=$BUILDPLATFORM golang:1.26.3-alpine AS builder
WORKDIR /src
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download
COPY . .
ARG APP_VERSION=V0.0.7_20260815
ARG TARGETOS=linux
ARG TARGETARCH=amd64
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build \
    -trimpath \
    -ldflags="-s -w -X github.com/PRINCEofAsgd/fluffy-cupcake/internal/version.Current=${APP_VERSION}" \
    -o /out/fluffy-cupcake ./cmd/server

# 运行阶段使用空白 Linux 镜像，仅保留静态服务二进制。
FROM scratch
WORKDIR /app
COPY --from=builder /out/fluffy-cupcake /app/fluffy-cupcake
USER 10001:10001
EXPOSE 4819
ENV APP_ADDR=0.0.0.0:4819 APP_MODE=release
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD ["/app/fluffy-cupcake", "healthcheck"]
ENTRYPOINT ["/app/fluffy-cupcake"]

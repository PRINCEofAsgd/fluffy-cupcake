// Package version 统一维护当前应用版本。
package version

// Current 是当前发布版本，格式为“主版本.次版本.修订版本_日期”；构建镜像时可通过 ldflags 覆盖。
var Current = "V0.0.6_20260815"

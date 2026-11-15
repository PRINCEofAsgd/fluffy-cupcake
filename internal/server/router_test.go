package server

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"

	"github.com/PRINCEofAsgd/fluffy-cupcake/internal/config"
	"github.com/PRINCEofAsgd/fluffy-cupcake/internal/version"
)

// TestHomepage 验证根路径返回 Loki Yin Homepage，并保留完整资料、资源加载门、来源说明和独立安全策略。
func TestHomepage(t *testing.T) {
	router := NewRouter(config.Config{Mode: "test"}, nil)
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/", nil)

	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("状态码 = %d，期望 %d", response.Code, http.StatusOK)
	}
	if location := response.Header().Get("Location"); location != "" {
		t.Fatalf("根路径不应重定向，实际 Location = %q", location)
	}
	body := response.Body.String()
	for _, expected := range []string{
		"<title>Loki Yin's Homepage</title>",
		`href="/favicon.png?v=` + version.Current + `"`,
		"页面参考 github.com/LeoninCS/Homepage 设计。",
		"/homepage/images/fora-hero-foreground.png",
	} {
		if !strings.Contains(body, expected) {
			t.Fatalf("Homepage 缺少 %q", expected)
		}
	}
	for _, privateContent := range []string{"献超前", "xianchaoqian@foxmail.com", "MiniMax Cloud Infra"} {
		if strings.Contains(body, privateContent) {
			t.Fatalf("Homepage 仍包含参考作者个人信息 %q", privateContent)
		}
	}
	if !strings.Contains(response.Header().Get("Content-Security-Policy"), "style-src 'self' 'unsafe-inline'") {
		t.Fatal("Homepage CSP 必须允许页面自身的动态样式")
	}
	assetPath := regexp.MustCompile(`/homepage/assets/[^"']+\.js`).FindString(body)
	if assetPath == "" {
		t.Fatal("Homepage 缺少 Vite JavaScript 产物引用")
	}
	assetResponse := httptest.NewRecorder()
	router.ServeHTTP(assetResponse, httptest.NewRequest(http.MethodGet, assetPath, nil))
	if assetResponse.Code != http.StatusOK {
		t.Fatalf("GET %s 状态码 = %d，期望 %d", assetPath, assetResponse.Code, http.StatusOK)
	}
	if !strings.Contains(assetResponse.Header().Get("Cache-Control"), "immutable") {
		t.Fatal("带哈希的 Homepage 资源必须使用 immutable 缓存")
	}
	assetBody := assetResponse.Body.String()
	for _, expected := range []string{
		"Loki Yin",
		"欢迎来到 Loki 的主页，这里记录了我的信息与经历，希望你能更好地了解我。",
		"给伴侣的点滴纪念",
		"可向我申请内测账号",
		"4³ ❖ Blind National Top 30",
		"3³ ❖ Blind sub60",
		"HOK Jungle National Top 50",
		"211 · CS | Go · Backend | AI Infra",
		"CS 大四 · 中国矿业大学(北京) · 北京",
		"中国矿业大学(北京) 计算机科学与技术 · 本科大四在读",
		"后端研发 · AI Infra · 游戏服务端",
		"Go · C++ · 分布式架构 · 云原生",
		"https://github.com/PRINCEofAsgd",
		"lokiyin4819@foxmail.com",
		"Yyg1061",
		"douyin.com/user/MS4wLjABAAAAqMBHTKsqMfKY5EldZgzCmvx6l3x-4sawM42QNPFn03LAB2oveG0wDDgPJNnDZNTa",
		"文安智能",
		"环界云计算",
		"实习经历",
		"开源经历",
		"Go 后端研发",
		"AIOT CLOUD / GO BACKEND",
		"ISSUE DETECTION / BUG FIX",
		"Sealos 开源项目 Contributor",
		"负责问题发现与复现、bug 定位与修复。",
		"完善应用删除时的关联资源清理与 Service Bind 新旧模型兼容链路。",
		"Service Bind 模型兼容",
		"用真实工程经历，铺叙出成长道路。",
		"从企业后端研发到云原生开源协作",
		"我的项目 < - > 也是我的小心愿",
		"Reactor Web",
		"Fluffy Cupcake",
		"My Dock",
		"Card Game",
		"https://music.163.com/#/artist?id=95001856",
		"matcha-cake-screen-wide.jpg",
		"tech-stack",
		"go.svg",
		"docker.svg",
		"kubernetes.svg",
		"experience/vion-tech.svg",
		"experience/vionvision-wordmark.svg",
	} {
		if !strings.Contains(assetBody, expected) {
			t.Fatalf("Homepage JavaScript 产物缺少公开资料 %q", expected)
		}
	}
	for _, removed := range []string{
		"奖项成果",
		"#competition",
		"Project 01",
	} {
		if strings.Contains(assetBody, removed) {
			t.Fatalf("Homepage JavaScript 产物仍包含已删除内容 %q", removed)
		}
	}
	// 项目标签需直接使用项目名，并按用户指定的 2、1、3、4 顺序呈现。
	projectOrder := []string{"Fluffy Cupcake", "Reactor Web", "My Dock", "Card Game"}
	previousIndex := -1
	for _, projectName := range projectOrder {
		projectIndex := strings.Index(assetBody, projectName)
		if projectIndex <= previousIndex {
			t.Fatalf("Homepage 项目顺序不正确，%q 的位置 = %d，前一项目位置 = %d", projectName, projectIndex, previousIndex)
		}
		previousIndex = projectIndex
	}
	homepagePathResponse := httptest.NewRecorder()
	router.ServeHTTP(homepagePathResponse, httptest.NewRequest(http.MethodGet, "/homepage/", nil))
	if homepagePathResponse.Code != http.StatusNotFound {
		t.Fatalf("旧 Homepage 页面路径状态码 = %d，期望 %d", homepagePathResponse.Code, http.StatusNotFound)
	}
}

// TestYanliliPage 验证页面、按钮文案和安全响应头均正确返回。
func TestYanliliPage(t *testing.T) {
	router := NewRouter(config.Config{Mode: "test"}, nil)
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/yanlili", nil)

	router.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("状态码 = %d，期望 %d", response.Code, http.StatusOK)
	}
	if !strings.Contains(response.Body.String(), "按按钮，想你+1") {
		t.Fatal("页面缺少点击后显示的目标文案")
	}
	if strings.Contains(response.Body.String(), "/assets/miss-pop.mp3") {
		t.Fatal("页面不应再加载 MP3，点击音效应由 Web Audio 合成")
	}
	if !strings.Contains(response.Body.String(), `href="/favicon.png?v=`+version.Current+`"`) {
		t.Fatal("想念页面缺少全站统一 PNG 标签图标")
	}
	if strings.Contains(response.Body.String(), "/assets/miss-button.png") || strings.Contains(response.Body.String(), "loki-yin-avatar.png") {
		t.Fatal("想念页面仍引用被替换的旧标签图标")
	}
	if !strings.Contains(response.Body.String(), `id="login-button"`) {
		t.Fatal("页面缺少独立登录入口")
	}
	if !strings.Contains(response.Body.String(), `id="companion-button"`) {
		t.Fatal("页面缺少未登录可见的陪伴绑定入口")
	}
	for _, expected := range []string{"我想ta", "每日想念", "最近想念", `id="direction-switch"`, `id="details-button"`} {
		if !strings.Contains(response.Body.String(), expected) {
			t.Fatalf("页面缺少新想念记录交互 %q", expected)
		}
	}
	if strings.Contains(response.Body.String(), `id="app-panel" class="app-panel" aria-label="想念按钮与陪伴绑定" hidden`) {
		t.Fatal("未登录状态不应隐藏主按钮面板")
	}
	if !strings.Contains(response.Body.String(), `id="stats-panel"`) || !strings.Contains(response.Body.String(), `id="stats-panel" class="stats-panel" aria-labelledby="stats-title" hidden`) {
		t.Fatal("数据库统计区域应默认隐藏并由登录状态控制")
	}
	contentSecurityPolicy := response.Header().Get("Content-Security-Policy")
	if contentSecurityPolicy == "" {
		t.Fatal("页面缺少 Content-Security-Policy 响应头")
	}
	if !strings.Contains(contentSecurityPolicy, "img-src 'self' blob:") {
		t.Fatal("页面 CSP 必须允许旧浏览器通过 blob URL 读取用户选择的二维码图片")
	}
}

// TestHealthAndAsset 验证健康检查与嵌入式静态资源可以访问。
func TestHealthAndAsset(t *testing.T) {
	router := NewRouter(config.Config{Mode: "test"}, nil)

	for _, testCase := range []struct {
		path        string
		contentType string
	}{
		{path: "/healthz", contentType: "application/json"},
		{path: "/favicon.png", contentType: "image/png"},
		{path: "/favicon.ico", contentType: "image/png"},
		{path: "/assets/miss-button.gif", contentType: "image/gif"},
		{path: "/assets/miss-pop.mp3", contentType: "audio/mpeg"},
		{path: "/assets/app.js", contentType: "text/javascript"},
		{path: "/assets/qrdecoder.js", contentType: "text/javascript"},
		{path: "/homepage/images/profile-placeholder.svg", contentType: "image/svg+xml"},
		{path: "/homepage/images/loki-yin-avatar.png", contentType: "image/png"},
		{path: "/homepage/images/github-mark.svg", contentType: "image/svg+xml"},
		{path: "/homepage/images/sealos.svg", contentType: "image/svg+xml"},
		{path: "/homepage/images/vion-tech.png", contentType: "image/png"},
		{path: "/homepage/images/matcha-cake-screen-wide.jpg", contentType: "image/jpeg"},
		{path: "/homepage/images/tech-stack/go.svg", contentType: "image/svg+xml"},
		{path: "/homepage/images/tech-stack/cpp.svg", contentType: "image/svg+xml"},
		{path: "/homepage/images/tech-stack/kafka.svg", contentType: "image/svg+xml"},
		{path: "/homepage/images/tech-stack/nginx.svg", contentType: "image/svg+xml"},
		{path: "/homepage/images/tech-stack/mysql.svg", contentType: "image/svg+xml"},
		{path: "/homepage/images/tech-stack/redis.svg", contentType: "image/svg+xml"},
		{path: "/homepage/images/tech-stack/prometheus.svg", contentType: "image/svg+xml"},
		{path: "/homepage/images/tech-stack/loki.svg", contentType: "image/svg+xml"},
		{path: "/homepage/images/tech-stack/kubernetes.svg", contentType: "image/svg+xml"},
		{path: "/homepage/images/tech-stack/docker.svg", contentType: "image/svg+xml"},
		{path: "/homepage/images/experience/vion-tech.svg", contentType: "image/svg+xml"},
		{path: "/homepage/images/experience/vionvision-wordmark.svg", contentType: "image/svg+xml"},
	} {
		response := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodGet, testCase.path, nil)
		router.ServeHTTP(response, request)

		if response.Code != http.StatusOK {
			t.Fatalf("GET %s 状态码 = %d，期望 %d", testCase.path, response.Code, http.StatusOK)
		}
		if !strings.Contains(response.Header().Get("Content-Type"), testCase.contentType) {
			t.Fatalf("GET %s Content-Type = %q", testCase.path, response.Header().Get("Content-Type"))
		}
	}
}

// TestSiteIconFallback 验证显式 PNG 地址与浏览器默认 ICO 回退返回同一份全站图标。
func TestSiteIconFallback(t *testing.T) {
	router := NewRouter(config.Config{Mode: "test"}, nil)
	responses := make([][]byte, 0, 2)
	for _, iconPath := range []string{"/favicon.png", "/favicon.ico"} {
		response := httptest.NewRecorder()
		router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, iconPath, nil))
		if response.Code != http.StatusOK {
			t.Fatalf("GET %s 状态码 = %d，期望 %d", iconPath, response.Code, http.StatusOK)
		}
		responses = append(responses, response.Body.Bytes())
	}
	if !bytes.Equal(responses[0], responses[1]) {
		t.Fatal("/favicon.png 与 /favicon.ico 未返回同一份图标")
	}
}

// TestQrScannerImplementation 防止摄像头首帧退出、相册 CSP 冲突和反色识别能力回退。
func TestQrScannerImplementation(t *testing.T) {
	router := NewRouter(config.Config{Mode: "test"}, nil)
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/assets/app.js", nil)

	router.ServeHTTP(response, request)
	body := response.Body.String()
	for _, expected := range []string{
		"requestVideoFrameCallback",
		"loadeddata",
		"scheduleQrScan(generation)",
		"createImageBitmap",
		`inversionAttempts: "attemptBoth"`,
		"decodeQrSource",
		"URL.revokeObjectURL",
	} {
		if !strings.Contains(body, expected) {
			t.Fatalf("app.js 缺少稳健二维码识别实现 %q", expected)
		}
	}
	if strings.Contains(body, `readyState < 2) return`) {
		t.Fatal("摄像头首帧未就绪时不得直接退出整个扫码循环")
	}
	if strings.Contains(body, `inversionAttempts: "dontInvert"`) {
		t.Fatal("二维码识别不得禁用反色尝试")
	}
}

// TestClickSoundImplementation 防止页面退回会互相打断的单实例 MP3 播放方式。
func TestClickSoundImplementation(t *testing.T) {
	router := NewRouter(config.Config{Mode: "test"}, nil)
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/assets/app.js", nil)

	router.ServeHTTP(response, request)

	body := response.Body.String()
	for _, expected := range []string{"AudioContext", "createOscillator", "createDynamicsCompressor"} {
		if !strings.Contains(body, expected) {
			t.Fatalf("app.js 缺少可并发 Web Audio 实现 %q", expected)
		}
	}
	if !strings.Contains(body, "该功能需登录后使用") {
		t.Fatal("app.js 缺少未登录绑定的明确提示")
	}
	if !strings.Contains(body, "ta想我") || !strings.Contains(body, "direction=mine") && !strings.Contains(body, "statsDirection") {
		t.Fatal("app.js 缺少双向记录切换逻辑")
	}
	for _, expected := range []string{"confirmUnbindRequest", "prompts.every", "confirm_inactive", "inactive_confirmation_required"} {
		if !strings.Contains(body, expected) {
			t.Fatalf("app.js 缺少统一解绑确认逻辑 %q", expected)
		}
	}
	for _, expected := range []string{"取消解绑申请", "拒绝解绑", "unbind-cancel", "unbind-reject", "unbind_status"} {
		if !strings.Contains(body, expected) {
			t.Fatalf("app.js 缺少解绑申请撤销/拒绝交互 %q", expected)
		}
	}
	if strings.Contains(body, "direct-unbind") || strings.Contains(body, "unbind-direct") {
		t.Fatal("app.js 不应暴露第二条直接解绑路线")
	}
	if strings.Contains(body, "new Audio(") {
		t.Fatal("app.js 不应再创建单实例 MP3 音频")
	}
}

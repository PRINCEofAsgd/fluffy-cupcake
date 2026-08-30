// 本脚本统一构建可由 Gin 嵌入的 Homepage 静态产物。
const fs = require("fs");
const path = require("path");
const { spawnSync } = require("child_process");

const root = path.resolve(__dirname, "..");
const distDir = path.join(root, "dist");

fs.rmSync(distDir, { recursive: true, force: true });

const result = spawnSync(
  process.execPath,
  [path.join(root, "node_modules", "vite", "bin", "vite.js"), "build"],
  {
    cwd: root,
    stdio: "inherit",
    env: process.env,
  },
);

if (result.status !== 0) {
  process.exit(result.status || 1);
}

const indexHtml = path.join(distDir, "index.html");
if (!fs.existsSync(indexHtml)) {
  console.error("Homepage 构建失败：dist/index.html 不存在");
  process.exit(1);
}

console.log("Homepage 静态资源已构建到 dist/");

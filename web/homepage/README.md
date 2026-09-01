# Homepage 前端

本目录保存根路径个人 Homepage 的 React/Vite 源码和静态资源。本次按用户说明已取得参考仓库作者批准，视觉与交互参考 `https://github.com/LeoninCS/Homepage`；加载页固定显示“页面参考 github.com/LeoninCS/Homepage 设计。”。

## 目录职责

- `src/main.jsx`：公开资料、工作与项目经历、爱好、整页资源加载门、滚动动效和各页面区块。
- `src/styles.css`：森林首屏、单层全屏照片、原作者经历控制台卡片、社媒轨道和响应式样式。
- `src/scrollTuning.js`：桌面与移动端滚动参数。
- `public/favicon.png`：Homepage、`/yanlili` 与浏览器默认回退共用的全站 PNG 图标。
- `public/homepage/`：以 `/homepage/*` 提供的字体、装饰图、Loki Yin 头像、首屏宽幅背景、经历品牌 SVG、自托管技术栈 Logo、占位图和社交分享图。
- `scripts/build-homepage.cjs`：构建并校验 `dist/index.html`。
- `dist/`：由 Vite 生成并交给 Go `embed.FS` 的产物。

## 资料状态

当前已填入 Loki Yin 的姓名、头像、主页标题、浏览器标签标题、基础 SEO 元数据、个人信息、两段工作经历、四个项目经历、四项爱好，以及 GitHub、邮箱、微信、抖音和网易云音乐联系方式。首屏资料卡使用 GitHub、Sealos 和 Vion 标识，单一背景层加载从用户照片重新取景的 16:9 宽幅图片；10 枚自托管技术栈 SVG 以低透明度紧凑托盘错落漂浮。实习与开源经历直接复用原作者的 MiniMax 与 Sealos 控制台卡片，项目导航直接使用项目名；爱好配图、博客与社交分享图文案继续使用明确占位，页面不展示奖项区。

## 填写资料

替换资料时应同时检查页面正文、SEO 元数据和社交分享图，不能只修改首屏：

1. 四项爱好的正式配图与更完整说明。
2. 博客或其他需要展示的公开社媒链接。
3. 社交分享图希望使用的主标题和副标题。

未提供的信息继续使用明确的“待填写”占位，不填入推测数据。

## 开发命令

```bash
npm ci
npm run dev
npm run build
```

生产构建由仓库根目录 Dockerfile 的 Node 22 阶段完成；本地 `make homepage-build` 会生成同样的 `dist`。

Vite 开发、预览与生产页面入口统一为默认路径 `/`；`/homepage/*` 仅作为静态资源命名空间，避免和 `/yanlili` 页面使用的 `/assets/*` 冲突。

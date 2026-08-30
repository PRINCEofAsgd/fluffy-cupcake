# Homepage 前端

本目录保存根路径个人 Homepage 的 React/Vite 源码和静态资源。本次按用户说明已取得参考仓库作者批准，视觉与交互参考 `https://github.com/LeoninCS/Homepage`；加载页固定显示“页面参考 github.com/LeoninCS/Homepage 设计。”。

## 目录职责

- `src/main.jsx`：资料数据、整页资源加载门、滚动动效和各页面区块。
- `src/styles.css`：森林首屏、预览卡片、叙事卡片、社媒轨道和响应式样式。
- `src/scrollTuning.js`：桌面与移动端滚动参数。
- `public/favicon.png`：Homepage、`/yanlili` 与浏览器默认回退共用的全站 PNG 图标。
- `public/homepage/`：以 `/homepage/*` 提供的字体、装饰图、Loki Yin 头像、占位图和社交分享图。
- `scripts/build-homepage.cjs`：构建并校验 `dist/index.html`。
- `dist/`：由 Vite 生成并交给 Go `embed.FS` 的产物。

## 资料状态

当前已填入 Loki Yin 的姓名、头像、主页标题、浏览器标签标题、三组个人信息、学校与学生身份、工作地、职业/技术方向，以及 GitHub、邮箱、微信、抖音联系方式。首屏资料卡使用 GitHub、Sealos 和 Vion 标识；经历、项目、兴趣、博客、SEO 元数据与社交分享图文案继续使用明确占位，页面不展示奖项区。

## 填写资料

替换资料时应同时检查页面正文、SEO 元数据和社交分享图，不能只修改首屏：

1. 最多两段代表经历：组织、角色、起止时间、职责、成果和配图。
2. 最多四个代表项目：名称、简介、技术栈、个人职责、成果、流程节点、仓库/演示链接。
3. 最多四项兴趣爱好：描述、投入程度、代表经历、关键词和配图。
4. 博客或其他需要展示的公开社媒链接。
5. SEO 标题、简介、关键词，以及社交分享图希望使用的主标题和副标题。

未提供的信息继续使用明确的“待填写”占位，不填入推测数据。

## 开发命令

```bash
npm ci
npm run dev
npm run build
```

生产构建由仓库根目录 Dockerfile 的 Node 22 阶段完成；本地 `make homepage-build` 会生成同样的 `dist`。

Vite 开发、预览与生产页面入口统一为默认路径 `/`；`/homepage/*` 仅作为静态资源命名空间，避免和 `/yanlili` 页面使用的 `/assets/*` 冲突。

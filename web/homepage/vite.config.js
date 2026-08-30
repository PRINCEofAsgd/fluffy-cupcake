import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';

// Homepage 以站点根路径作为开发和预览入口，构建资源继续隔离在 /homepage/ 命名空间中。
export default defineConfig({
  base: '/',
  plugins: [react()],
  build: {
    assetsDir: 'homepage/assets',
    modulePreload: {
      polyfill: false,
    },
  },
});

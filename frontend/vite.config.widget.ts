import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import tailwindcss from '@tailwindcss/vite'
import { fileURLToPath, URL } from 'node:url'

export default defineConfig({
  plugins: [
    vue({
      customElement: /\.ce\.vue$/,
    }),
    tailwindcss(),
  ],
  resolve: {
    alias: {
      '@': fileURLToPath(new URL('./src', import.meta.url)),
    },
  },
  define: {
    'process.env.NODE_ENV': JSON.stringify('production'),
  },
  build: {
    lib: {
      entry: fileURLToPath(new URL('./src/widget/widget-main.ts', import.meta.url)),
      name: 'ArgusWidget',
      fileName: (format) => `argus-widget.${format}.js`,
    },
    cssFileName: 'argus-widget',
    outDir: 'dist-widget',
    emptyOutDir: true,
    minify: 'oxc',
    rollupOptions: {
      // Vue 全量 bundle，不 externalize
      output: {
        // UMD 全局变量
        globals: {},
      },
    },
  },
})

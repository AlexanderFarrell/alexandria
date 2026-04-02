import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import { VitePWA } from 'vite-plugin-pwa'
import { resolve } from 'path'

export default defineConfig({
  plugins: [
    vue(),
    VitePWA({
      strategies: 'generateSW',
      registerType: 'autoUpdate',
      injectRegister: 'auto',
      filename: 'sw.js',
      manifest: {
        name: 'Alexandria',
        short_name: 'Alexandria',
        description: 'Self-hosted reading library — your books, anywhere.',
        theme_color: '#c8a96e',
        background_color: '#0f0f13',
        display: 'standalone',
        orientation: 'any',
        scope: '/',
        start_url: '/library',
        id: '/library',
        icons: [
          { src: '/pwa-64x64.png',            sizes: '64x64',   type: 'image/png' },
          { src: '/pwa-192x192.png',           sizes: '192x192', type: 'image/png' },
          { src: '/pwa-512x512.png',           sizes: '512x512', type: 'image/png', purpose: 'any' },
          { src: '/maskable-icon-512x512.png', sizes: '512x512', type: 'image/png', purpose: 'maskable' },
        ],
      },
      workbox: {
        globPatterns: ['**/*.{js,css,html,ico,svg,woff,woff2}'],
        globIgnores: ['**/*.epub', '**/*.pdf'],
        runtimeCaching: [
          {
            urlPattern: /^\/api\/v1\/books(\?.*)?$/,
            handler: 'NetworkFirst',
            options: {
              cacheName: 'api-books',
              networkTimeoutSeconds: 5,
              expiration: { maxEntries: 10, maxAgeSeconds: 86400 },
              cacheableResponse: { statuses: [0, 200] },
            },
          },
          {
            urlPattern: /^\/api\/v1\/books\/[^/]+$/,
            handler: 'NetworkFirst',
            options: {
              cacheName: 'api-book-detail',
              networkTimeoutSeconds: 5,
              expiration: { maxEntries: 50, maxAgeSeconds: 86400 },
              cacheableResponse: { statuses: [0, 200] },
            },
          },
          {
            urlPattern: /^\/api\/v1\/books\/[^/]+\/cover$/,
            handler: 'StaleWhileRevalidate',
            options: {
              cacheName: 'book-covers',
              expiration: { maxEntries: 100, maxAgeSeconds: 604800 },
              cacheableResponse: { statuses: [0, 200] },
            },
          },
          { urlPattern: /^\/api\/v1\/auth\//, handler: 'NetworkOnly' },
          { urlPattern: /^\/api\/v1\/books\/[^/]+\/reader\/manifest$/, handler: 'NetworkOnly' },
          { urlPattern: /^\/api\/v1\/books\/[^/]+\/reader\/sections\/[^/]+$/, handler: 'NetworkOnly' },
          { urlPattern: /^\/api\/v1\/books\/[^/]+\/reader\/assets\//, handler: 'NetworkOnly' },
          { urlPattern: /^\/api\/v1\/books\/[^/]+\/content$/, handler: 'NetworkOnly' },
        ],
        navigateFallback: 'index.html',
        navigateFallbackDenylist: [/^\/api\//, /^\/health$/, /^\/readyz$/],
        skipWaiting: true,
        clientsClaim: true,
      },
      devOptions: { enabled: false },
    }),
  ],
  resolve: {
    alias: {
      '@': resolve(__dirname, 'src'),
    },
  },
  server: {
    port: 5173,
    proxy: {
      '/api': {
        target: 'http://localhost:8080',
        changeOrigin: true,
      },
    },
  },
  build: {
    outDir: 'dist',
    emptyOutDir: true,
  },
})

/// <reference types="vitest/config" />
import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite'
import path from 'node:path'

// Admin backend target for the dev proxy.
const API = process.env.VITE_API_PROXY ?? 'http://localhost:8091'

export default defineConfig({
  plugins: [react(), tailwindcss()],
  resolve: {
    alias: {
      '@': path.resolve(import.meta.dirname, 'src'),
      '@common': path.resolve(import.meta.dirname, '../common/src'),
    },
    dedupe: ['react', 'react-dom'],
  },
  server: {
    port: 5174,
    fs: { allow: [path.resolve(import.meta.dirname), path.resolve(import.meta.dirname, '../common')] },
    proxy: {
      '/api': { target: API, changeOrigin: true },
      '/openapi': { target: API, changeOrigin: true },
    },
  },
  test: { environment: 'jsdom', globals: true, passWithNoTests: true },
})

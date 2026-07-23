import react from '@vitejs/plugin-react';
import { defineConfig } from 'vite';

// https://vite.dev/config/
export default defineConfig({
  plugins: [react()],
  test: {
    environment: 'jsdom',
    globals: true,
    setupFiles: ['./src/test-setup.js'],
    // Exclude Playwright test files to avoid conflicts
    exclude: [
      '**/node_modules/**',
      '**/dist/**',
      '**/tests/**', // Exclude Playwright e2e tests
      '**/.{idea,git,cache,output,temp}/**',
      '**/{karma,rollup,webpack,vite,vitest,jest,ava,babel,nyc,cypress,tsup,build}.config.*',
    ],
  },
  build: {
    rollupOptions: {
      output: {
        // Function form (required by Vite 8 / Rolldown; the rollup object form
        // is not supported and fails with "manualChunks is not a function").
        manualChunks(id) {
          if (id.includes('@mui/icons-material')) return 'mui-icons';
          if (
            id.includes('@mui/material') ||
            id.includes('@mui/system') ||
            id.includes('@emotion/react') ||
            id.includes('@emotion/styled')
          ) {
            return 'mui';
          }
          if (id.includes('react-router')) return 'react-router';
          if (/[\\/]node_modules[\\/]react(-dom)?[\\/]/.test(id)) {
            return 'react-vendor';
          }
          if (
            /[\\/]src[\\/]components[\\/](Auth|Database|General|Notification)Settings\.jsx$/.test(
              id
            )
          ) {
            return 'settings';
          }
        },
      },
    },
    // Increase chunk size warning limit to 750kb
    chunkSizeWarningLimit: 750,
    // Enable minification and compression with esbuild (faster than terser)
    minify: 'esbuild',
    // Configure esbuild to drop console statements in production
    esbuild: {
      drop: ['console', 'debugger'],
      // Enable tree shaking
      treeShaking: true,
    },
    // Enable source maps for better debugging (disabled for production)
    sourcemap: false,
    // Optimize CSS
    cssMinify: true,
    // Optimize assets
    assetsInlineLimit: 4096,
  },
});

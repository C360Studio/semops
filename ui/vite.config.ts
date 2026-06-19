import { sveltekit } from '@sveltejs/kit/vite';
import { defineConfig } from 'vite';

export default defineConfig({
  plugins: [sveltekit()],
  build: {
    rollupOptions: {
      output: {
        manualChunks(id) {
          if (
            id.includes('/node_modules/@deck.gl/') ||
            id.includes('/node_modules/@luma.gl/') ||
            id.includes('/node_modules/@math.gl/') ||
            id.includes('/node_modules/probe.gl/')
          ) {
            return 'cop-map-renderer';
          }
          if (id.includes('/node_modules/maplibre-gl/')) {
            return 'cop-maplibre';
          }
        }
      }
    }
  },
  server: {
    proxy: {
      '/api': 'http://127.0.0.1:8088',
      '/healthz': 'http://127.0.0.1:8088'
    }
  },
  test: {
    include: ['src/**/*.test.ts']
  }
});

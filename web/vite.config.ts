import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';
import tailwindcss from '@tailwindcss/vite';

export default defineConfig(() => {
  const apiTarget = process.env.VITE_API_TARGET ?? 'http://localhost:10000';

  return {
    plugins: [react(), tailwindcss()],
    server: {
      host: '0.0.0.0',
      port: 10001,
      allowedHosts: ['.llassingan.web.id'],
      proxy: {
        '/api': {
          target: apiTarget,
          changeOrigin: true,
          proxyTimeout: 0,
          configure: (proxy) => {
            proxy.on('proxyReq', (proxyReq, req) => {
              if (req.url?.includes('/events')) {
                proxyReq.setHeader('Connection', 'keep-alive');
              }
            });
            proxy.on('proxyRes', (proxyRes, req) => {
              if (req.url?.includes('/events')) {
                proxyRes.headers['cache-control'] = 'no-cache, no-transform';
                proxyRes.headers['x-accel-buffering'] = 'no';
                proxyRes.headers['connection'] = 'keep-alive';
              }
            });
          },
        },
      },
    },
  };
});

import { defineConfig } from 'vitest/config';

// We deliberately do not use @vitejs/plugin-react here. The version pinned in
// the project (^2.x) injects a Fast Refresh preamble that requires the dev
// server to be running and breaks under vitest. Esbuild handles JSX
// compilation on its own, which is sufficient for component tests.
export default defineConfig({
  esbuild: {
    jsx: 'automatic',
  },
  test: {
    environment: 'happy-dom',
    include: ['src/**/*.{test,spec}.{ts,tsx}'],
    setupFiles: ['./src/test-setup.ts'],
    globals: true,
  },
});

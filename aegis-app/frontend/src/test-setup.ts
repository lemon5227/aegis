import '@testing-library/jest-dom/vitest';
import { afterEach } from 'vitest';
import { cleanup } from '@testing-library/react';

// Cleanup the rendered DOM between tests so happy-dom state doesn't leak.
afterEach(() => {
  cleanup();
});

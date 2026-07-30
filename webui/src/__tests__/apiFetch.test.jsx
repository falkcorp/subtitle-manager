// file: webui/src/__tests__/apiFetch.test.jsx
// version: 1.0.0
// guid: 7f3b2c58-41d9-4e06-8a75-1c9e0d4b6f32

import { beforeEach, describe, expect, test, vi } from 'vitest';

/**
 * Load a fresh copy of the api module with window.location.pathname set.
 *
 * The base path is resolved once, at module load, so it cannot be changed
 * afterwards — the module has to be re-imported for each case.
 */
async function loadApi(pathname) {
  vi.resetModules();
  Object.defineProperty(window, 'location', {
    value: { ...window.location, pathname },
    writable: true,
    configurable: true,
  });
  return import('../services/api.js');
}

describe('apiFetch', () => {
  beforeEach(() => {
    global.fetch = vi.fn(() => Promise.resolve({ ok: true }));
  });

  // The defect this exists to prevent: components that called fetch('/api/...')
  // directly sent no installation prefix, so under a server base_url of /sm the
  // browser requested /api/tags rather than /sm/api/tags. That path has no
  // route, so the SPA catch-all answered with index.html and a 200 — the
  // component then failed parsing HTML as JSON, or rendered nothing at all.
  test('prefixes the installation base path', async () => {
    const { apiFetch } = await loadApi('/sm/settings');
    await apiFetch('/api/tags');
    expect(fetch).toHaveBeenCalledWith('/sm/api/tags');
  });

  test('is a no-op when served from the root', async () => {
    const { apiFetch } = await loadApi('/settings');
    await apiFetch('/api/tags');
    expect(fetch).toHaveBeenCalledWith('/api/tags');
  });

  test('leaves absolute URLs alone', async () => {
    const { apiFetch } = await loadApi('/sm/settings');
    await apiFetch('http://example.test/api/tags');
    expect(fetch).toHaveBeenCalledWith('http://example.test/api/tags');
  });

  // Always passing a second argument would turn fetch(url) into
  // fetch(url, undefined) — behaviourally identical, but it changes what a spy
  // records and broke callers' existing assertions for no reason.
  test('preserves call arity', async () => {
    const { apiFetch } = await loadApi('/settings');
    await apiFetch('/api/tags');
    expect(fetch.mock.calls[0]).toHaveLength(1);

    fetch.mockClear();
    await apiFetch('/api/tags', { method: 'POST' });
    expect(fetch.mock.calls[0]).toHaveLength(2);
  });
});

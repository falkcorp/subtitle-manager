// file: webui/src/__tests__/SettingsProviderWiring.test.jsx
// version: 1.0.0
// guid: 7a5c02be-4f31-4d9a-8c60-b1e93d478f52
// last-edited: 2026-08-12

// Settings → Providers was completely non-functional: the page PATCHed
// /api/providers/{name} and POSTed /api/providers/{name}/config, neither of
// which is a mounted route. Both were absorbed by the /api/providers/ subtree
// handler, and because each call site was an `if (response.ok)` with no
// `else`, every failure was reported to the operator as success.
//
// These tests assert the observable effect — the request that goes out, and
// what the UI shows afterwards. They deliberately do not assert on
// console.error: React reports a throw inside an event handler as an unhandled
// error that Vitest does not count as a failure, so a test written that way
// passes with the bug reinstated.

import '@testing-library/jest-dom/vitest';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { beforeEach, describe, expect, test, vi } from 'vitest';
import ProviderConfigDialog from '../components/ProviderConfigDialog.jsx';

// opensubtitles requires api_key and user_agent. Without them isValid() keeps
// the Save button disabled and the click is a no-op — which would make every
// assertion in this file vacuously pass. The fixture has to be genuinely valid.
const PROVIDER = {
  name: 'opensubtitles',
  config: { api_key: 'k', user_agent: 'ua' },
};

vi.mock('../services/api.js', () => ({
  apiService: {
    get: vi.fn(() =>
      Promise.resolve({ ok: true, json: () => Promise.resolve([]) })
    ),
  },
  getBasePath: () => '',
}));

describe('provider configuration dialog', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  // The dialog used to call onSave and onClose back to back, so a rejected
  // save looked exactly like an accepted one.
  test('stays open when the save fails', async () => {
    const onSave = vi.fn(() => Promise.resolve(false));
    const onClose = vi.fn();

    render(
      <ProviderConfigDialog
        open
        provider={PROVIDER}
        onClose={onClose}
        onSave={onSave}
      />
    );

    fireEvent.click(screen.getByRole('button', { name: /save/i }));

    await waitFor(() => expect(onSave).toHaveBeenCalled());
    expect(onClose).not.toHaveBeenCalled();
  });

  test('closes when the save succeeds', async () => {
    const onSave = vi.fn(() => Promise.resolve(true));
    const onClose = vi.fn();

    render(
      <ProviderConfigDialog
        open
        provider={PROVIDER}
        onClose={onClose}
        onSave={onSave}
      />
    );

    fireEvent.click(screen.getByRole('button', { name: /save/i }));

    await waitFor(() => expect(onClose).toHaveBeenCalled());
  });

  // A handler that reports nothing must keep the old close-on-save behaviour,
  // so this change cannot silently wedge other callers' dialogs open.
  test('closes when the handler reports nothing', async () => {
    const onSave = vi.fn(() => undefined);
    const onClose = vi.fn();

    render(
      <ProviderConfigDialog
        open
        provider={PROVIDER}
        onClose={onClose}
        onSave={onSave}
      />
    );

    fireEvent.click(screen.getByRole('button', { name: /save/i }));

    await waitFor(() => expect(onClose).toHaveBeenCalled());
  });
});

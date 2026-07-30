// file: webui/src/__tests__/ProviderConfigDialog.test.jsx
// version: 1.0.0
// guid: e2f3a4b5-c6d7-4e8f-9012-3456789abcde

import { beforeEach, describe, expect, test, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import { useState } from 'react';
import ProviderCard from '../components/ProviderCard.jsx';
import ProviderConfigDialog from '../components/ProviderConfigDialog.jsx';

// Mock apiService to avoid network requests
vi.mock('../services/api.js', () => ({
  apiService: {
    get: vi.fn(() =>
      Promise.resolve({ ok: true, json: () => Promise.resolve([]) })
    ),
  },
}));

function Wrapper() {
  const [open, setOpen] = useState(false);
  return (
    <>
      <ProviderCard
        provider={{
          name: 'opensubtitles',
          displayName: 'OpenSubtitles',
          enabled: false,
        }}
        onToggle={() => {}}
        onConfigure={() => setOpen(true)}
      />
      <ProviderConfigDialog
        open={open}
        provider={{ name: 'opensubtitles' }}
        onClose={() => setOpen(false)}
        onSave={() => setOpen(false)}
      />
    </>
  );
}

describe('Provider configuration', () => {
  test('opens dialog when configuring provider', () => {
    render(<Wrapper />);
    fireEvent.click(screen.getByText('OpenSubtitles'));
    expect(screen.getByText(/configure provider/i)).toBeInTheDocument();
  });
});

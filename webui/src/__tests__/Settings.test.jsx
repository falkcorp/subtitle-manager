// file: webui/src/__tests__/Settings.test.jsx
import '@testing-library/jest-dom/vitest';
import {
  act,
  fireEvent,
  render,
  screen,
  waitFor,
} from '@testing-library/react';
import { afterEach, beforeEach, describe, expect, test, vi } from 'vitest';
import { MemoryRouter } from 'react-router-dom';
import Settings from '../Settings.jsx';

// Settings calls useNavigate, which throws outside a Router. Rendering it bare
// failed every test in this file with "useNavigate() may be used only in the
// context of a <Router> component" — the component grew routing after these
// tests were written.
const renderSettings = () =>
  render(
    <MemoryRouter>
      <Settings />
    </MemoryRouter>
  );

// Mock the API service
vi.mock('../services/api.js', () => ({
  apiService: {
    get: vi.fn(),
    post: vi.fn(),
  },
  getBasePath: () => '',
}));

describe('Settings component', () => {
  const originalFetch = global.fetch;

  beforeEach(async () => {
    vi.clearAllMocks();

    global.fetch = vi.fn(url => {
      if (url === '/api/providers/available') {
        return Promise.resolve({
          ok: true,
          json: () =>
            Promise.resolve([{ name: 'opensubtitles' }, { name: 'subscene' }]),
        });
      }
      return Promise.resolve({ ok: true, json: () => Promise.resolve({}) });
    });

    // Get the mocked apiService
    const { apiService } = await import('../services/api.js');

    // Setup default mocks
    apiService.get.mockImplementation(url => {
      if (url === '/api/config') {
        return Promise.resolve({
          ok: true,
          json: () => Promise.resolve({ address: '0.0.0.0' }),
        });
      }
      if (url === '/api/providers') {
        return Promise.resolve({
          ok: true,
          json: () =>
            Promise.resolve([
              { name: 'opensubtitles', enabled: true, config: {} },
              { name: 'subscene', enabled: false, config: {} },
            ]),
        });
      }
      // ProviderConfigDialog loads this through apiService.get, not fetch, so
      // mocking global.fetch for it (as this file also does) never reached the
      // component — it fell through to the catch-all below and received an
      // object where an array was expected.
      if (url === '/api/providers/available') {
        return Promise.resolve({
          ok: true,
          json: () =>
            Promise.resolve([{ name: 'opensubtitles' }, { name: 'subscene' }]),
        });
      }
      return Promise.resolve({
        ok: true,
        json: () => Promise.resolve({}),
      });
    });

    apiService.post.mockResolvedValue({
      ok: true,
      json: () => Promise.resolve({}),
    });
  });

  afterEach(() => {
    global.fetch = originalFetch;
  });
  test('loads settings and renders tabs', async () => {
    await act(async () => {
      renderSettings();
    });

    // Wait for the settings to load and tabs to appear
    await waitFor(() => {
      expect(screen.getByText('Settings')).toBeInTheDocument();
      expect(screen.getByText('General')).toBeInTheDocument();
      expect(screen.getByText('Providers')).toBeInTheDocument();
    });

    // Verify API calls were made to load config
    const { apiService } = await import('../services/api.js');

    await waitFor(() => {
      expect(apiService.get).toHaveBeenCalledWith('/api/config');
      expect(apiService.get).toHaveBeenCalledWith('/api/providers');
    });

    // Click the General tab to verify it can be selected
    await act(async () => {
      fireEvent.click(screen.getByRole('tab', { name: /General/i }));
    });

    // Verify the General tab becomes selected
    await waitFor(() => {
      const generalTab = screen.getByRole('tab', { name: /General/i });
      expect(generalTab).toHaveAttribute('aria-selected', 'true');
    });
  });

  test('shows only enabled providers', async () => {
    await act(async () => {
      renderSettings();
    });

    await waitFor(() => {
      expect(screen.getByText('OpenSubtitles')).toBeInTheDocument();
    });

    expect(screen.queryByText('Subscene')).toBeNull();
  });

  test('add provider dialog lists all providers', async () => {
    await act(async () => {
      renderSettings();
    });

    const addButton = screen.getByText('Add Provider');
    await act(async () => {
      fireEvent.click(addButton);
    });

    await waitFor(() => {
      expect(screen.getByRole('dialog')).toBeInTheDocument();
    });

    const select = screen.getByRole('combobox');
    await act(async () => {
      fireEvent.mouseDown(select);
    });

    // All available providers should be listed
    expect(await screen.findByText('Subscene')).toBeInTheDocument();
  });

  test('import dialog allows entering Bazarr details', async () => {
    await act(async () => {
      renderSettings();
    });

    const importButton = screen.getByText('Import from Bazarr');
    await act(async () => {
      fireEvent.click(importButton);
    });

    await waitFor(() => {
      expect(screen.getByRole('dialog')).toBeInTheDocument();
    });

    expect(screen.getByLabelText('Bazarr URL')).toBeInTheDocument();
    expect(screen.getByLabelText('API Key')).toBeInTheDocument();
  });

  test('shows permission error when config fetch forbidden', async () => {
    const { apiService } = await import('../services/api.js');
    apiService.get.mockImplementationOnce(url => {
      if (url === '/api/config') {
        return Promise.resolve({ ok: false, status: 403 });
      }
      if (url === '/api/providers') {
        return Promise.resolve({ ok: true, json: () => Promise.resolve([]) });
      }
      return Promise.resolve({ ok: true, json: () => Promise.resolve({}) });
    });

    await act(async () => {
      renderSettings();
    });

    expect(await screen.findByText('Permission denied')).toBeInTheDocument();
  });
});

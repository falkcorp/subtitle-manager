// file: webui/src/__tests__/App.test.jsx
import '@testing-library/jest-dom/vitest';
import {
  act,
  fireEvent,
  render,
  screen,
  waitFor,
} from '@testing-library/react';
import { beforeEach, describe, expect, test, vi } from 'vitest';
import App from '../App.jsx';
import { apiService } from '../services/api.js';

// Mock the API service
vi.mock('../services/api.js', () => ({
  apiService: {
    checkBackendHealth: vi.fn(),
    get: vi.fn(),
    post: vi.fn(),
    put: vi.fn(),
    delete: vi.fn(),
  },
  getBasePath: () => '',
  // App calls apiFetch for the endpoints it has not migrated to apiService.
  // A module mock must export it too, or the component calls undefined.
  // Delegating to global.fetch keeps this file's existing fetch-based
  // assertions meaningful.
  apiFetch: (url, options) =>
    options === undefined ? fetch(url) : fetch(url, options),
}));

describe('App component', () => {
  beforeEach(() => {
    vi.clearAllMocks();

    // Clear localStorage
    localStorage.clear();

    // Setup default mocks
    global.fetch = vi.fn(() =>
      Promise.resolve({ ok: false, json: () => Promise.resolve({}) })
    );
  });

  test('shows login form when unauthenticated', async () => {
    // Mock backend available but not authenticated
    apiService.checkBackendHealth.mockResolvedValue(true);
    apiService.get.mockImplementation(url => {
      if (url === '/api/config') {
        return Promise.resolve({ ok: false });
      }
      if (url === '/api/setup/status') {
        return Promise.resolve({
          ok: true,
          json: () => Promise.resolve({ needed: false }),
        });
      }
      return Promise.resolve({ ok: false });
    });

    await act(async () => {
      render(<App />);
    });

    await waitFor(() => {
      // The login heading renders t('app.title'), so its text depends on
      // whether i18n has been initialised in the test environment — asserting
      // the translated string made this test a check on translation setup
      // rather than on the login form. The subtitle beneath it is a literal
      // and identifies the same screen unambiguously.
      expect(
        screen.getByText(/sign in to access your subtitle management/i)
      ).toBeInTheDocument();
    });
  });

  test('successful login renders dashboard', async () => {
    // Mock backend available but not authenticated initially
    apiService.checkBackendHealth.mockResolvedValue(true);
    apiService.get.mockImplementation(url => {
      if (url === '/api/config') {
        return Promise.resolve({ ok: false });
      }
      if (url === '/api/setup/status') {
        return Promise.resolve({
          ok: true,
          json: () => Promise.resolve({ needed: false }),
        });
      }
      return Promise.resolve({ ok: false });
    });

    await act(async () => {
      render(<App />);
    });

    // Wait for login form
    await waitFor(() => {
      expect(screen.getByText('Sign In')).toBeInTheDocument();
    });

    // Mock successful login
    fetch.mockResolvedValueOnce({ ok: true });

    await act(async () => {
      fireEvent.click(screen.getByText('Sign In'));
    });

    await waitFor(() =>
      expect(fetch).toHaveBeenLastCalledWith('/api/login', expect.any(Object))
    );
  });

  test('pins sidebar and persists state', async () => {
    // Mock backend available and authenticated
    apiService.checkBackendHealth.mockResolvedValue(true);
    apiService.get.mockImplementation(url => {
      if (url === '/api/config') {
        return Promise.resolve({ ok: true });
      }
      if (url === '/api/setup/status') {
        return Promise.resolve({
          ok: true,
          json: () => Promise.resolve({ needed: false }),
        });
      }
      return Promise.resolve({ ok: true });
    });

    await act(async () => {
      render(<App />);
    });

    // Wait for dashboard to load
    await waitFor(() => screen.getByLabelText('open drawer'));

    await act(async () => {
      fireEvent.click(screen.getByLabelText('open drawer'));
    });

    const pinButton = await screen.findByRole('button', {
      name: 'Pin Sidebar',
    });

    await act(async () => {
      fireEvent.click(pinButton);
    });

    expect(localStorage.getItem('sidebarPinned')).toBe('true');

    await act(async () => {
      fireEvent.click(screen.getByRole('button', { name: 'Unpin Sidebar' }));
    });

    expect(localStorage.getItem('sidebarPinned')).toBe('false');
  });
});

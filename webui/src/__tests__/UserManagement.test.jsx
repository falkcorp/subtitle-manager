// file: webui/src/__tests__/UserManagement.test.jsx
// version: 1.1.0
// guid: 184acee4-a24c-4363-8893-b3d5394f8e5c
import '@testing-library/jest-dom/vitest';
import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { beforeEach, describe, expect, test, vi } from 'vitest';
import UserManagement from '../UserManagement.jsx';

// Mock the API service, not global.fetch.
//
// UserManagement was migrated to apiService.users.* and no longer calls fetch
// directly, so mocking fetch left the component hitting the real service while
// the assertions waited for a fetch call that never came. The tests failed on
// a timeout with a misleading DOM dump rather than saying "wrong seam".
vi.mock('../services/api.js', () => ({
  apiService: {
    users: {
      list: vi.fn(),
      create: vi.fn(),
      update: vi.fn(),
      resetPassword: vi.fn(),
    },
  },
  getBasePath: () => '',
}));

const { apiService } = await import('../services/api.js');

/** jsonOk builds the fetch-like response shape apiService resolves to. */
const jsonOk = data => ({ ok: true, json: () => Promise.resolve(data) });

describe('UserManagement component', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  test('displays usernames from API', async () => {
    apiService.users.list.mockResolvedValueOnce(
      jsonOk([
        { id: '1', username: 'alice', email: 'a@example.com', role: 'admin' },
      ])
    );

    render(<UserManagement />);
    await waitFor(() => expect(apiService.users.list).toHaveBeenCalled());
    expect(await screen.findByText('alice')).toBeInTheDocument();
  });

  test('shows fallback when username missing', async () => {
    apiService.users.list.mockResolvedValueOnce(
      jsonOk([
        { id: '42', email: 'no-name@example.com', role: 'user', active: true },
      ])
    );

    render(<UserManagement />);

    await waitFor(() => expect(apiService.users.list).toHaveBeenCalled());

    expect(await screen.findByText('42')).toBeInTheDocument();
  });

  test('opens editor dialog when add user clicked', async () => {
    apiService.users.list.mockResolvedValueOnce(jsonOk([]));

    render(<UserManagement />);
    await waitFor(() => expect(apiService.users.list).toHaveBeenCalled());

    fireEvent.click(screen.getByText('Add User'));

    const dialog = await screen.findByRole('dialog');
    expect(dialog).toBeInTheDocument();
    expect(screen.getByText('Save')).toBeInTheDocument();
  });
});

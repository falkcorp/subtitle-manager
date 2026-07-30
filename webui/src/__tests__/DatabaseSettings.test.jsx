// file: webui/src/__tests__/DatabaseSettings.test.jsx
import '@testing-library/jest-dom/vitest';
import { render, screen, waitFor } from '@testing-library/react';
import { beforeEach, describe, expect, test, vi } from 'vitest';
import DatabaseSettings from '../components/DatabaseSettings.jsx';

// Mock the API service, not global.fetch: this component calls
// apiService.database.* and never touches fetch, so a fetch mock left the
// assertions waiting on a call that could not happen.
vi.mock('../services/api.js', () => ({
  apiService: {
    database: {
      getInfo: vi.fn(),
      getStats: vi.fn(),
      backup: vi.fn(),
      optimize: vi.fn(),
    },
  },
  getBasePath: () => '',
}));

const { apiService } = await import('../services/api.js');

const jsonOk = data => ({ ok: true, json: () => Promise.resolve(data) });

describe('DatabaseSettings component', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    apiService.database.getInfo.mockResolvedValue(
      jsonOk({
        type: 'postgresql',
        version: '13',
        size: 1048576,
        path: '/db',
        connected: true,
      })
    );
    apiService.database.getStats.mockResolvedValue(
      jsonOk({
        totalRecords: 100,
        users: 5,
        downloads: 20,
        mediaItems: 30,
        lastBackup: '2024-05-01T00:00:00Z',
      })
    );
    apiService.database.backup.mockResolvedValue(jsonOk({}));
    apiService.database.optimize.mockResolvedValue(jsonOk({}));
  });

  test('displays database info from API', async () => {
    render(<DatabaseSettings config={{}} onSave={() => {}} backendAvailable />);

    await waitFor(() => expect(apiService.database.getInfo).toHaveBeenCalled());

    expect(await screen.findByText('postgresql')).toBeInTheDocument();
    expect(await screen.findByText('Connected')).toBeInTheDocument();
  });

  test('shows warning when backend unavailable', () => {
    render(
      <DatabaseSettings
        config={{}}
        onSave={() => {}}
        backendAvailable={false}
      />
    );

    expect(
      screen.getByText(/Backend service is not available/i)
    ).toBeInTheDocument();
  });
});

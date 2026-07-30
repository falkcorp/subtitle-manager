// file: webui/src/__tests__/ConfigEditor.test.jsx
import '@testing-library/jest-dom';
import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { vi } from 'vitest';
import ConfigEditor from '../ConfigEditor.jsx';

// Mock the API service, not global.fetch: ConfigEditor uses apiService.get and
// apiService.post, so a fetch mock observed nothing.
vi.mock('../services/api.js', () => ({
  apiService: { get: vi.fn(), post: vi.fn() },
  getBasePath: () => '',
}));

const { apiService } = await import('../services/api.js');

describe('ConfigEditor component', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    apiService.get.mockResolvedValue({
      ok: true,
      json: () => Promise.resolve({ test_key: 'value' }),
    });
    apiService.post.mockResolvedValue({ ok: true });
  });

  test('loads config and saves updates', async () => {
    render(<ConfigEditor />);
    await waitFor(() =>
      expect(apiService.get).toHaveBeenCalledWith('/api/config')
    );
    // MUI puts data-testid on the TextField wrapper, not the inner control,
    // so .value on it is undefined. Query the textarea itself.
    const editor = screen
      .getByTestId('config-editor')
      .querySelector('textarea');
    expect(editor.value).toContain('test_key');
    fireEvent.change(editor, { target: { value: 'test_key: new' } });
    fireEvent.click(screen.getByText('Save'));
    await waitFor(() =>
      expect(apiService.post).toHaveBeenCalledWith(
        '/api/config',
        expect.anything()
      )
    );
  });
});

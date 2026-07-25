// file: webui/src/__tests__/Wanted.test.jsx
import '@testing-library/jest-dom';
import {
  fireEvent,
  render,
  screen,
  waitFor,
  within,
} from '@testing-library/react';
import { expect, vi } from 'vitest';
import Wanted from '../Wanted.jsx';

// The wanted list holds media files missing subtitles (Bazarr's model), not
// bookmarked subtitle download URLs.
//
// The previous test asserted a GET `/api/search?provider=...` call the
// component had already stopped making, so it was failing on the base branch
// before any of these changes.
describe('Wanted component', () => {
  beforeEach(() => {
    vi.restoreAllMocks();
    global.fetch = vi.fn();
  });

  const wantedItem = {
    id: 'item-1',
    path: '/media/Some.Movie.2021.1080p.mkv',
    languages: ['en'],
    status: 'pending',
    retry_count: 0,
    max_retries: 3,
  };

  test('renders monitored media on mount', async () => {
    fetch.mockResolvedValueOnce({
      ok: true,
      json: () => Promise.resolve([wantedItem]),
    });

    render(<Wanted />);

    // Shown by basename, with its language and status. Scoped to the list
    // item: 'EN' also appears in the page's language selector.
    const name = await screen.findByText('Some.Movie.2021.1080p.mkv');
    const row = within(name.closest('li'));
    expect(row.getByText('EN')).toBeInTheDocument();
    expect(row.getByText('pending')).toBeInTheDocument();
  });

  test('monitoring a file posts the media path, not a subtitle URL', async () => {
    fetch.mockResolvedValueOnce({ ok: true, json: () => Promise.resolve([]) });
    render(<Wanted />);
    await waitFor(() => expect(fetch).toHaveBeenCalledWith('/api/wanted'));

    fireEvent.change(screen.getByPlaceholderText('/path/to/movie.mkv'), {
      target: { value: '/media/Another.Movie.mkv' },
    });

    // POST, then the component reloads the list.
    fetch.mockResolvedValueOnce({ ok: true, json: () => Promise.resolve({}) });
    fetch.mockResolvedValueOnce({ ok: true, json: () => Promise.resolve([]) });

    fireEvent.click(screen.getByText('Monitor this file'));

    await waitFor(() =>
      expect(fetch).toHaveBeenCalledWith(
        '/api/wanted',
        expect.objectContaining({
          method: 'POST',
          body: JSON.stringify({
            path: '/media/Another.Movie.mkv',
            languages: ['en'],
          }),
        })
      )
    );
  });

  test('removing sends the media path', async () => {
    fetch.mockResolvedValueOnce({
      ok: true,
      json: () => Promise.resolve([wantedItem]),
    });
    render(<Wanted />);
    await screen.findByText('Some.Movie.2021.1080p.mkv');

    fetch.mockResolvedValueOnce({ ok: true, json: () => Promise.resolve({}) });
    fireEvent.click(screen.getByTestId('DeleteIcon').closest('button'));

    await waitFor(() =>
      expect(fetch).toHaveBeenCalledWith(
        '/api/wanted',
        expect.objectContaining({
          method: 'DELETE',
          body: JSON.stringify({ path: wantedItem.path }),
        })
      )
    );
  });
});

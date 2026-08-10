// file: webui/src/__tests__/MediaLibraryBrowseShape.test.jsx
// version: 1.0.0
// guid: 3c8b17e0-5d24-49f6-a0b1-6e93f4c2a785
// last-edited: 2026-08-10

import '@testing-library/jest-dom';
import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import { beforeEach, describe, expect, test, vi } from 'vitest';
import MediaLibrary from '../MediaLibrary.jsx';

// These fixtures are transcribed from what pkg/webserver actually marshals for
// GET /api/library/browse — see the MediaItem struct in pkg/webserver/server.go,
// which encodes `isDirectory`. The pre-existing MediaLibraryBulk.test.jsx feeds
// this component `is_dir`, a key the server has never sent, so that suite and
// the component agreed with each other while both disagreed with production:
// every directory was filtered out and the page rendered completely empty.
//
// Assertions here are on observable effects — what is on screen, and which
// requests were issued. Never on console.error: React reports a throw inside an
// event handler as an unhandled error that Vitest does not count as a failure,
// so a test that spies on the error log passes happily with the bug reinstated.

const ROOT_ITEMS = [
  {
    name: 'Breaking Bad',
    path: '/media/Breaking Bad',
    isDirectory: true,
    modTime: '2026-08-01T00:00:00Z',
  },
  {
    name: 'Movie.mkv',
    path: '/media/Movie.mkv',
    isDirectory: false,
    size: 1024,
    modTime: '2026-08-01T00:00:00Z',
  },
  // A sidecar subtitle sitting next to the video, exactly as the server lists
  // it. The grid hides it, so "Select all files" must hide it too.
  {
    name: 'Movie.en.srt',
    path: '/media/Movie.en.srt',
    isDirectory: false,
    size: 12,
    modTime: '2026-08-01T00:00:00Z',
  },
];

const SEASON_ITEMS = [
  {
    name: 'S01E01.mkv',
    path: '/media/Breaking Bad/S01E01.mkv',
    isDirectory: false,
    size: 2048,
    modTime: '2026-08-01T00:00:00Z',
  },
];

/** Every /api/library/browse URL the component has requested, in order. */
let browseCalls = [];

function mockApi() {
  browseCalls = [];
  global.fetch = vi.fn(url => {
    if (url.startsWith('/api/library/browse')) {
      browseCalls.push(url);
      const requested = decodeURIComponent(
        new URLSearchParams(url.split('?')[1]).get('path') || '/'
      );
      const items = requested.includes('Breaking Bad')
        ? SEASON_ITEMS
        : ROOT_ITEMS;
      return Promise.resolve({
        ok: true,
        json: () => Promise.resolve({ items }),
      });
    }
    if (url === '/api/language-profiles') {
      return Promise.resolve({ ok: true, json: () => Promise.resolve([]) });
    }
    return Promise.resolve({ ok: true, json: () => Promise.resolve([]) });
  });
}

/** Paths requested from the browse endpoint, decoded. */
function browsedPaths() {
  return browseCalls.map(url =>
    decodeURIComponent(new URLSearchParams(url.split('?')[1]).get('path') || '')
  );
}

function renderLibrary() {
  return render(
    <MemoryRouter>
      <MediaLibrary />
    </MemoryRouter>
  );
}

describe('MediaLibrary against the real browse payload', () => {
  beforeEach(() => {
    mockApi();
  });

  test('renders a directory the server marked isDirectory', async () => {
    renderLibrary();

    // The root of a media tree is all directories. If these are filtered out
    // the page is blank and Mass Edit is unreachable.
    expect(await screen.findByText('Breaking Bad')).toBeInTheDocument();
  });

  test('Select all files skips directories and sidecar subtitles', async () => {
    renderLibrary();
    await screen.findByText('Movie.mkv');

    fireEvent.click(screen.getByRole('button', { name: /mass edit/i }));
    fireEvent.click(screen.getByRole('button', { name: /select all files/i }));

    // Only Movie.mkv is a media file. Selecting the directory or the .srt
    // attaches a language profile to something that cannot have one — the
    // observed symptom was "4 assigned" for two visible episodes.
    expect(await screen.findByText('1 selected')).toBeInTheDocument();
  });

  test('opening a directory browses that directory, not the previous one', async () => {
    renderLibrary();
    fireEvent.click(await screen.findByText('Breaking Bad'));

    // The item click used to call setCurrentPath(path) and then
    // loadCurrentDirectory() back to back. The second call closed over the
    // *previous* currentPath, so it refetched the directory being left and
    // whichever response landed last won the render.
    await waitFor(() =>
      expect(browsedPaths()).toContain('/media/Breaking Bad')
    );
    expect(browsedPaths().filter(path => path === '/')).toHaveLength(1);
  });

  test('a breadcrumb click navigates back to that directory', async () => {
    renderLibrary();
    fireEvent.click(await screen.findByText('Breaking Bad'));
    await screen.findByText('S01E01.mkv');

    // The breadcrumbs called loadDirectory(), which is defined nowhere in the
    // file, so every breadcrumb click threw a ReferenceError. Assert the
    // effect — that we actually navigate — rather than watching console.error,
    // which React does not surface to Vitest as a failure.
    fireEvent.click(screen.getByRole('button', { name: 'Home' }));

    expect(await screen.findByText('Movie.mkv')).toBeInTheDocument();
  });
});

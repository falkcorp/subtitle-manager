// file: webui/src/__tests__/MediaLibraryBulk.test.jsx
// version: 1.1.0
// guid: 9f4c1e73-2b8d-46a1-b0e5-7c3a9d5e1042
// last-edited: 2026-08-04

import '@testing-library/jest-dom';
import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import { beforeEach, describe, expect, test, vi } from 'vitest';
import MediaLibrary from '../MediaLibrary.jsx';

// Mass edit exists to assign a language profile to many files at once. The
// interesting behaviour is not that a button renders — it is *what request the
// component sends* and *what it does with a partial result*, because the
// endpoint reports per-item outcomes inside a 200 and an earlier version of
// this component swallowed bulk failures into console.error.

const FILES = [
  { name: 'A.mkv', path: '/media/A.mkv', is_dir: false, size: 1 },
  { name: 'B.mkv', path: '/media/B.mkv', is_dir: false, size: 1 },
  { name: 'Season 01', path: '/media/Season 01', is_dir: true },
];

const PROFILES = [
  { id: 'p1', name: 'English Only', is_default: true },
  { id: 'p2', name: 'Dual Audio', is_default: false },
];

/**
 * Route the component's mount-time fetches by URL.
 *
 * MediaLibrary issues browse / library-paths / language-profiles / tasks
 * concurrently, so ordered mockResolvedValueOnce chains are fragile here —
 * a dispatch keeps each test insensitive to that ordering.
 */
function mockApi(bulkHandler) {
  global.fetch = vi.fn((url, options) => {
    if (url.startsWith('/api/library/browse')) {
      return Promise.resolve({
        ok: true,
        json: () => Promise.resolve({ items: FILES }),
      });
    }
    if (url === '/api/language-profiles') {
      return Promise.resolve({
        ok: true,
        json: () => Promise.resolve(PROFILES),
      });
    }
    if (url === '/api/media/profiles/bulk') {
      return bulkHandler(url, options);
    }
    // library/paths, tasks, and anything else the component probes.
    return Promise.resolve({ ok: true, json: () => Promise.resolve([]) });
  });
}

function renderLibrary() {
  return render(
    <MemoryRouter>
      <MediaLibrary />
    </MemoryRouter>
  );
}

async function enterMassEditAndSelectBoth() {
  await screen.findByText('A.mkv');
  fireEvent.click(screen.getByRole('button', { name: /mass edit/i }));
  fireEvent.click(screen.getByLabelText('Select A.mkv'));
  fireEvent.click(screen.getByLabelText('Select B.mkv'));
}

describe('MediaLibrary mass edit', () => {
  beforeEach(() => {
    vi.restoreAllMocks();
  });

  test('posts every selected path to the bulk endpoint', async () => {
    const bulk = vi.fn(() =>
      Promise.resolve({
        ok: true,
        json: () =>
          Promise.resolve({
            profile_id: 'p2',
            succeeded: 2,
            failed: 0,
            results: [
              { media_id: '/media/A.mkv', ok: true },
              { media_id: '/media/B.mkv', ok: true },
            ],
          }),
      })
    );
    mockApi(bulk);
    renderLibrary();
    await enterMassEditAndSelectBoth();

    fireEvent.mouseDown(screen.getByLabelText('Language profile'));
    fireEvent.click(await screen.findByText('Dual Audio'));
    fireEvent.click(screen.getByRole('button', { name: /apply to 2/i }));

    await waitFor(() => expect(bulk).toHaveBeenCalled());
    const body = JSON.parse(bulk.mock.calls[0][1].body);
    expect(body.profile_id).toBe('p2');
    // Both selections must survive to the request. A set-to-array conversion
    // that dropped one would still look fine on screen.
    expect(body.media_ids.sort()).toEqual(['/media/A.mkv', '/media/B.mkv']);

    expect(await screen.findByText('2 assigned.')).toBeInTheDocument();
  });

  test('directories are not selectable', async () => {
    mockApi(() => Promise.reject(new Error('should not be called')));
    renderLibrary();
    await screen.findByText('A.mkv');
    fireEvent.click(screen.getByRole('button', { name: /mass edit/i }));

    // A directory has no profile assignment, so offering a checkbox that does
    // nothing would be worse than offering none.
    expect(screen.queryByLabelText('Select Season 01')).toBeNull();
  });

  test('select all survives the Library Paths tab, which lists no media', async () => {
    mockApi(() => Promise.reject(new Error('should not be called')));
    renderLibrary();
    await enterMassEditAndSelectBoth();
    expect(screen.getByText('2 selected')).toBeInTheDocument();

    // The toolbar is shown whenever mass edit is on, independent of the active
    // tab, so this tab is reachable with Select all on screen. getTabContent()
    // returns null here and an unguarded .filter throws.
    fireEvent.click(screen.getByRole('tab', { name: 'Library Paths' }));
    fireEvent.click(screen.getByRole('button', { name: /select all files/i }));

    // Starting from a non-empty selection is what makes this discriminating.
    // React reports a throw inside an event handler as an unhandled error, which
    // Vitest does not count as a test failure — so asserting "0 selected" from
    // an already-empty selection would pass whether the handler ran or threw.
    // Reaching 0 from 2 is only possible if it ran to completion.
    expect(screen.getByText('0 selected')).toBeInTheDocument();
  });

  test('a partial failure is reported and the failed items stay selected', async () => {
    mockApi(() =>
      Promise.resolve({
        ok: true,
        json: () =>
          Promise.resolve({
            profile_id: 'p1',
            succeeded: 1,
            failed: 1,
            results: [
              { media_id: '/media/A.mkv', ok: true },
              { media_id: '/media/B.mkv', ok: false, error: 'not found' },
            ],
          }),
      })
    );
    renderLibrary();
    await enterMassEditAndSelectBoth();
    fireEvent.click(screen.getByRole('button', { name: /apply to 2/i }));

    expect(
      await screen.findByText(/1 of 2 assigned .*1 failed and remain selected/i)
    ).toBeInTheDocument();
    // Retrying must not re-send the item that already succeeded.
    await waitFor(() =>
      expect(
        screen.getByRole('button', { name: /apply to 1/i })
      ).toBeInTheDocument()
    );
  });

  test('a whole-request rejection surfaces as an error, not a silent no-op', async () => {
    mockApi(() =>
      Promise.resolve({
        ok: false,
        status: 400,
        text: () => Promise.resolve('unknown profile'),
        json: () => Promise.resolve({}),
      })
    );
    renderLibrary();
    await enterMassEditAndSelectBoth();
    fireEvent.click(screen.getByRole('button', { name: /apply to 2/i }));

    expect(await screen.findByText(/unknown profile/i)).toBeInTheDocument();
  });
});

// file: webui/src/__tests__/MediaLibraryCombineSubs.test.jsx
// version: 1.0.0
// guid: 6a3f81d0-27b5-4c94-8e16-b0d75fc2a349
// last-edited: 2026-08-11

import '@testing-library/jest-dom';
import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import { beforeEach, describe, expect, test, vi } from 'vitest';
import MediaLibrary from '../MediaLibrary.jsx';

// Shape transcribed from pkg/webserver MediaItem/Subtitle: the browse endpoint
// already reports each media file's sidecars, the UI just never showed them.
const ITEMS = [
  {
    name: 'Pilot.mkv',
    path: '/media/Pilot.mkv',
    isDirectory: false,
    size: 1024,
    subtitles: [
      { language: 'English', path: '/media/Pilot.en.srt', format: 'srt' },
      { language: 'Spanish', path: '/media/Pilot.es.srt', format: 'srt' },
      { language: 'French', path: '/media/Pilot.fr.srt', format: 'srt' },
    ],
  },
  {
    name: 'NoSubs.mkv',
    path: '/media/NoSubs.mkv',
    isDirectory: false,
    size: 1024,
  },
];

let stackCalls = [];

function mockApi(stackResponse) {
  stackCalls = [];
  global.fetch = vi.fn((url, options) => {
    if (url.startsWith('/api/library/browse')) {
      return Promise.resolve({
        ok: true,
        json: () => Promise.resolve({ items: ITEMS }),
      });
    }
    if (url === '/api/subtitles/stack') {
      stackCalls.push(JSON.parse(options.body));
      return Promise.resolve(
        stackResponse || {
          ok: true,
          json: () => Promise.resolve({ output: '/media/Pilot.eo.srt' }),
        }
      );
    }
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

describe('MediaLibrary combine subtitles', () => {
  beforeEach(() => {
    vi.restoreAllMocks();
    mockApi();
  });

  test('lists the subtitles the server reported for a media file', async () => {
    renderLibrary();
    await screen.findByText('Pilot.mkv');

    expect(
      await screen.findByLabelText('Select subtitle English')
    ).toBeInTheDocument();
    expect(
      screen.getByLabelText('Select subtitle Spanish')
    ).toBeInTheDocument();
    expect(screen.getByLabelText('Select subtitle French')).toBeInTheDocument();
  });

  test('combining two subtitles posts them as primary and secondary', async () => {
    renderLibrary();
    await screen.findByText('Pilot.mkv');

    fireEvent.click(await screen.findByLabelText('Select subtitle English'));
    fireEvent.click(screen.getByLabelText('Select subtitle Spanish'));
    fireEvent.click(
      screen.getByRole('button', { name: /combine subtitles for Pilot\.mkv/i })
    );

    await waitFor(() => expect(stackCalls).toHaveLength(1));
    // Order matters: the primary language renders on top of the stacked cue.
    expect(stackCalls[0]).toEqual({
      primary: '/media/Pilot.en.srt',
      secondary: '/media/Pilot.es.srt',
    });
  });

  test('combine is only available once exactly two subtitles are selected', async () => {
    renderLibrary();
    await screen.findByText('Pilot.mkv');

    const button = await screen.findByRole('button', {
      name: /combine subtitles for Pilot\.mkv/i,
    });
    expect(button).toBeDisabled();

    fireEvent.click(screen.getByLabelText('Select subtitle English'));
    expect(button).toBeDisabled();

    fireEvent.click(screen.getByLabelText('Select subtitle Spanish'));
    expect(button).toBeEnabled();

    // A third selection is ambiguous — stacking combines exactly two.
    fireEvent.click(screen.getByLabelText('Select subtitle French'));
    expect(button).toBeDisabled();
  });

  test('a file with no subtitles offers no combine control', async () => {
    renderLibrary();
    await screen.findByText('NoSubs.mkv');

    expect(
      screen.queryByRole('button', {
        name: /combine subtitles for NoSubs\.mkv/i,
      })
    ).toBeNull();
  });

  test('a failed combine reports the error instead of failing silently', async () => {
    mockApi({
      ok: false,
      status: 400,
      text: () => Promise.resolve('cannot read secondary subtitle'),
      json: () => Promise.resolve({}),
    });
    renderLibrary();
    await screen.findByText('Pilot.mkv');

    fireEvent.click(await screen.findByLabelText('Select subtitle English'));
    fireEvent.click(screen.getByLabelText('Select subtitle Spanish'));
    fireEvent.click(
      screen.getByRole('button', { name: /combine subtitles for Pilot\.mkv/i })
    );

    // Silent failure is the recurring defect in this codebase's settings pages;
    // assert the user is told, not that console.error happened.
    expect(
      await screen.findByText(/cannot read secondary subtitle/i)
    ).toBeInTheDocument();
  });
});

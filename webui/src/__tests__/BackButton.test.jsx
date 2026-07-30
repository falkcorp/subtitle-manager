// file: webui/src/__tests__/BackButton.test.jsx
// version: 1.0.0
// guid: c1d2e3f4-a5b6-4c7d-8e9f-0123456789ab

import { beforeEach, describe, expect, test, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import BackButton from '../components/BackButton.jsx';
import { MemoryRouter } from 'react-router-dom';

const mockNavigate = vi.fn();

// vi.importActual is async, unlike jest's requireActual: the factory has to
// await it and spread the resolved module, or every real export (BrowserRouter
// here) is missing from the mock.
vi.mock('react-router-dom', async () => ({
  ...(await vi.importActual('react-router-dom')),
  useNavigate: () => mockNavigate,
}));

// Replacing window.history wholesale breaks BrowserRouter, which calls
// history.replaceState on mount — hence MemoryRouter here, which keeps its
// own history and does not touch the global. Only `length` needs stubbing,
// and it is read-only on the real History object, so a redefined property is
// still the way to set it.
const setHistoryLength = len =>
  Object.defineProperty(window, 'history', {
    value: { ...window.history, length: len },
    configurable: true,
  });

describe('BackButton', () => {
  test('navigates back when history exists', () => {
    setHistoryLength(2);
    render(<BackButton />, { wrapper: MemoryRouter });
    fireEvent.click(screen.getByRole('button', { name: /back/i }));
    expect(mockNavigate).toHaveBeenCalledWith(-1);
  });

  test('navigates home when no history', () => {
    mockNavigate.mockClear();
    setHistoryLength(1);
    render(<BackButton />, { wrapper: MemoryRouter });
    fireEvent.click(screen.getByRole('button', { name: /back/i }));
    expect(mockNavigate).toHaveBeenCalledWith('/');
  });
});

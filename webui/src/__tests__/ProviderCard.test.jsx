// file: webui/src/__tests__/ProviderCard.test.jsx
import '@testing-library/jest-dom/vitest';
import { fireEvent, render, screen } from '@testing-library/react';
import { vi } from 'vitest';
import ProviderCard from '../components/ProviderCard.jsx';

describe('ProviderCard component', () => {
  test('opens configuration when card is clicked', () => {
    const onConfigure = vi.fn();
    const provider = {
      name: 'opensubtitles',
      displayName: 'OpenSubtitles',
      enabled: true,
    };
    render(
      <ProviderCard
        provider={provider}
        onToggle={vi.fn()}
        onConfigure={onConfigure}
      />
    );

    fireEvent.click(screen.getByText('OpenSubtitles'));
    expect(onConfigure).toHaveBeenCalledTimes(1);
  });

  test('does not open configuration when toggling provider', () => {
    const onConfigure = vi.fn();
    const onToggle = vi.fn();
    const provider = {
      name: 'opensubtitles',
      displayName: 'OpenSubtitles',
      enabled: false,
    };
    render(
      <ProviderCard
        provider={provider}
        onToggle={onToggle}
        onConfigure={onConfigure}
      />
    );

    // MUI's Switch renders role="switch", not "checkbox"; the component
    // uses a Switch for the enabled toggle.
    fireEvent.click(screen.getByRole('switch'));
    expect(onToggle).toHaveBeenCalled();
    expect(onConfigure).not.toHaveBeenCalled();
  });
});

// file: webui/src/__tests__/NotificationSettings.test.jsx
// version: 1.0.0
// guid: 6a2d81f4-90c7-4b35-8e10-5f7c40b9e2d8
// last-edited: 2026-07-26
import '@testing-library/jest-dom';
import { fireEvent, render, screen } from '@testing-library/react';
import { describe, expect, test, vi } from 'vitest';
import NotificationSettings from '../components/NotificationSettings.jsx';

// The key namespace is the whole point of these tests. POST /api/config passes
// each key straight to viper.Set, and the Go runtime reads `notifications.*`.
// Saving bare keys therefore wrote settings nowhere anything read them, and the
// page reported success while never delivering a notification.
describe('NotificationSettings config keys', () => {
  test('saves keys under the notifications namespace', () => {
    const onSave = vi.fn();
    render(<NotificationSettings config={{}} onSave={onSave} />);

    fireEvent.click(screen.getByText('Save Notification Settings'));

    expect(onSave).toHaveBeenCalledTimes(1);
    const saved = onSave.mock.calls[0][0];

    // Every key namespaced; none left bare, where the runtime cannot see it.
    const bare = Object.keys(saved).filter(
      k => !k.startsWith('notifications.')
    );
    expect(bare).toEqual([]);
    expect(saved).toHaveProperty('notifications.discord_webhook');
    expect(saved).toHaveProperty('notifications.smtp_host');
  });

  test('reads namespaced values', () => {
    render(
      <NotificationSettings
        config={{
          notifications: { email_enabled: true, smtp_host: 'mail.example.com' },
        }}
        onSave={vi.fn()}
      />
    );
    expect(screen.getByDisplayValue('mail.example.com')).toBeInTheDocument();
  });

  // A config written by an older build has the bare key. Ignoring it would make
  // a working configuration look empty, inviting the user to overwrite it.
  test('falls back to bare keys from older builds', () => {
    render(
      <NotificationSettings
        config={{ email_enabled: true, smtp_host: 'legacy.example.com' }}
        onSave={vi.fn()}
      />
    );
    expect(screen.getByDisplayValue('legacy.example.com')).toBeInTheDocument();
  });
});

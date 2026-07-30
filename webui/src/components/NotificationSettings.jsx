// file: webui/src/components/NotificationSettings.jsx
import {
  Delete as DeleteIcon,
  Chat as DiscordIcon,
  Email as EmailIcon,
  Notifications as PushIcon,
  Telegram as TelegramIcon,
  Send as TestIcon,
  Webhook as WebhookIcon,
} from '@mui/icons-material';
import {
  Alert,
  Box,
  Button,
  Card,
  CardActions,
  CardContent,
  Chip,
  CircularProgress,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  FormControl,
  FormControlLabel,
  Grid,
  InputLabel,
  MenuItem,
  Select,
  Switch,
  TextField,
  Typography,
} from '@mui/material';
import { useEffect, useState } from 'react';
import { apiFetch } from '../services/api.js';

/**
 * NotificationSettings displays card-based configuration for various
 * notification methods including email, Discord, Telegram, push and
 * generic webhooks.
 *
 * @param {Object} props - Component properties
 * @param {Object} props.config - Existing configuration values
 * @param {Function} props.onSave - Callback invoked with updated values
 * @param {boolean} [props.backendAvailable=true] - Whether the backend API can be reached
 * @returns {JSX.Element} Notification settings form
 */
export default function NotificationSettings({
  config,
  onSave,
  backendAvailable = true,
}) {
  // Email notifications
  const [emailEnabled, setEmailEnabled] = useState(false);
  const [smtpHost, setSmtpHost] = useState('');
  const [smtpPort, setSmtpPort] = useState(587);
  const [smtpUsername, setSmtpUsername] = useState('');
  const [smtpPassword, setSmtpPassword] = useState('');
  const [smtpFrom, setSmtpFrom] = useState('');
  const [smtpTo, setSmtpTo] = useState('');
  const [smtpTLS, setSmtpTLS] = useState(true);

  // Discord notifications
  const [discordEnabled, setDiscordEnabled] = useState(false);
  const [discordWebhook, setDiscordWebhook] = useState('');
  const [discordUsername, setDiscordUsername] = useState('Subtitle Manager');
  const [discordAvatar, setDiscordAvatar] = useState('');

  // Telegram notifications
  const [telegramEnabled, setTelegramEnabled] = useState(false);
  const [telegramToken, setTelegramToken] = useState('');
  const [telegramChatId, setTelegramChatId] = useState('');

  // Push notifications
  const [pushEnabled, setPushEnabled] = useState(false);
  const [pushoverToken, setPushoverToken] = useState('');
  const [pushoverUser, setPushoverUser] = useState('');

  // Webhook notifications
  const [webhookEnabled, setWebhookEnabled] = useState(false);
  const [webhookUrl, setWebhookUrl] = useState('');
  const [webhookMethod, setWebhookMethod] = useState('POST');
  const [webhookHeaders, setWebhookHeaders] = useState('');

  // Dialog states
  const [testDialog, setTestDialog] = useState({
    open: false,
    type: '',
    loading: false,
  });
  const [resetDialog, setResetDialog] = useState(false);

  // Error state for notification errors
  const [error, setError] = useState(null);

  // Load configuration values when provided
  useEffect(() => {
    if (config) {
      // Email
      setEmailEnabled(config.email_enabled || false);
      setSmtpHost(config.smtp_host || '');
      setSmtpPort(config.smtp_port || 587);
      setSmtpUsername(config.smtp_username || '');
      setSmtpPassword(config.smtp_password || '');
      setSmtpFrom(config.smtp_from || '');
      setSmtpTo(config.smtp_to || '');
      setSmtpTLS(config.smtp_tls !== false);

      // Discord
      setDiscordEnabled(config.discord_enabled || false);
      setDiscordWebhook(config.discord_webhook || '');
      setDiscordUsername(config.discord_username || 'Subtitle Manager');
      setDiscordAvatar(config.discord_avatar || '');

      // Telegram
      setTelegramEnabled(config.telegram_enabled || false);
      setTelegramToken(config.telegram_token || '');
      setTelegramChatId(config.telegram_chat_id || '');

      // Push
      setPushEnabled(config.push_enabled || false);
      setPushoverToken(config.pushover_token || '');
      setPushoverUser(config.pushover_user || '');

      // Webhook
      setWebhookEnabled(config.webhook_enabled || false);
      setWebhookUrl(config.webhook_url || '');
      setWebhookMethod(config.webhook_method || 'POST');
      setWebhookHeaders(config.webhook_headers || '');
    }
  }, [config]);

  // Persist updated configuration
  const handleSave = () => {
    const newConfig = {
      email_enabled: emailEnabled,
      smtp_host: smtpHost,
      smtp_port: parseInt(smtpPort, 10),
      smtp_username: smtpUsername,
      smtp_password: smtpPassword,
      smtp_from: smtpFrom,
      smtp_to: smtpTo,
      smtp_tls: smtpTLS,

      discord_enabled: discordEnabled,
      discord_webhook: discordWebhook,
      discord_username: discordUsername,
      discord_avatar: discordAvatar,

      telegram_enabled: telegramEnabled,
      telegram_token: telegramToken,
      telegram_chat_id: telegramChatId,

      push_enabled: pushEnabled,
      pushover_token: pushoverToken,
      pushover_user: pushoverUser,

      webhook_enabled: webhookEnabled,
      webhook_url: webhookUrl,
      webhook_method: webhookMethod,
      webhook_headers: webhookHeaders,
    };

    onSave(newConfig);
  };

  // Send a test notification for the provided type
  const testNotification = async type => {
    setTestDialog({ open: true, type, loading: true });
    try {
      const response = await apiFetch(`/api/notifications/test/${type}`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          message: 'Test notification from Subtitle Manager',
        }),
      });

      if (response.ok) {
        alert(`${type} notification sent successfully!`);
      } else {
        const err = await response.text();
        alert(`Failed to send ${type} notification: ${err}`);
      }
    } catch {
      setError('Failed to update notification settings');
    } finally {
      setTestDialog({ open: false, type: '', loading: false });
    }
  };

  // Reset all notification settings to defaults
  const resetAllNotifications = () => {
    setEmailEnabled(false);
    setSmtpHost('');
    setSmtpPort(587);
    setSmtpUsername('');
    setSmtpPassword('');
    setSmtpFrom('');
    setSmtpTo('');
    setSmtpTLS(true);

    setDiscordEnabled(false);
    setDiscordWebhook('');
    setDiscordUsername('Subtitle Manager');
    setDiscordAvatar('');

    setTelegramEnabled(false);
    setTelegramToken('');
    setTelegramChatId('');

    setPushEnabled(false);
    setPushoverToken('');
    setPushoverUser('');

    setWebhookEnabled(false);
    setWebhookUrl('');
    setWebhookMethod('POST');
    setWebhookHeaders('');

    setResetDialog(false);
    alert('All notification settings reset to defaults');
  };

  return (
    <Box sx={{ maxWidth: 1200 }}>
      <Box
        sx={{
          display: 'flex',
          justifyContent: 'space-between',
          alignItems: 'center',
          mb: 3,
        }}
      >
        <Typography variant="h6">Notification Settings</Typography>
        <Button
          variant="outlined"
          color="error"
          startIcon={<DeleteIcon />}
          onClick={() => setResetDialog(true)}
          disabled={!backendAvailable}
        >
          Reset All Notifications
        </Button>
      </Box>

      {!backendAvailable && (
        <Alert severity="warning" sx={{ mb: 3 }}>
          Backend service is not available. Notification settings cannot be
          modified.
        </Alert>
      )}

      {error && (
        <Alert severity="error" sx={{ mb: 3 }} onClose={() => setError(null)}>
          {/* Display backend error from notification update */}
          {error}
        </Alert>
      )}

      <Grid container spacing={3}>
        {/* Email Notifications */}
        <Grid item xs={12} md={6}>
          <Card sx={{ height: '100%' }}>
            <CardContent>
              <Box sx={{ display: 'flex', alignItems: 'center', mb: 2 }}>
                <EmailIcon sx={{ mr: 1, color: 'primary.main' }} />
                <Typography variant="h6" color="primary">
                  Email (SMTP)
                </Typography>
                <Box sx={{ flexGrow: 1 }} />
                <Switch
                  checked={emailEnabled}
                  onChange={e => setEmailEnabled(e.target.checked)}
                  disabled={!backendAvailable}
                />
              </Box>

              <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>
                Send notifications via email using SMTP.
              </Typography>

              {emailEnabled && (
                <Box sx={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
                  <Grid container spacing={2}>
                    <Grid item xs={8}>
                      <TextField
                        fullWidth
                        label="SMTP Host"
                        value={smtpHost}
                        onChange={e => setSmtpHost(e.target.value)}
                        placeholder="smtp.gmail.com"
                        size="small"
                        disabled={!backendAvailable}
                      />
                    </Grid>
                    <Grid item xs={4}>
                      <TextField
                        fullWidth
                        label="Port"
                        type="number"
                        value={smtpPort}
                        onChange={e => setSmtpPort(e.target.value)}
                        size="small"
                        disabled={!backendAvailable}
                      />
                    </Grid>
                  </Grid>

                  <TextField
                    fullWidth
                    label="Username"
                    value={smtpUsername}
                    onChange={e => setSmtpUsername(e.target.value)}
                    size="small"
                    disabled={!backendAvailable}
                  />

                  <TextField
                    fullWidth
                    label="Password"
                    type="password"
                    value={smtpPassword}
                    onChange={e => setSmtpPassword(e.target.value)}
                    size="small"
                    disabled={!backendAvailable}
                  />

                  <TextField
                    fullWidth
                    label="From Address"
                    value={smtpFrom}
                    onChange={e => setSmtpFrom(e.target.value)}
                    placeholder="subtitles@example.com"
                    size="small"
                    disabled={!backendAvailable}
                  />

                  <TextField
                    fullWidth
                    label="To Address"
                    value={smtpTo}
                    onChange={e => setSmtpTo(e.target.value)}
                    placeholder="admin@example.com"
                    size="small"
                    disabled={!backendAvailable}
                  />

                  <FormControlLabel
                    control={
                      <Switch
                        checked={smtpTLS}
                        onChange={e => setSmtpTLS(e.target.checked)}
                        disabled={!backendAvailable}
                      />
                    }
                    label="Use TLS/SSL"
                  />
                </Box>
              )}
            </CardContent>

            {emailEnabled && (
              <CardActions>
                <Button
                  size="small"
                  startIcon={<TestIcon />}
                  onClick={() => testNotification('email')}
                  disabled={
                    !backendAvailable || !smtpHost || !smtpFrom || !smtpTo
                  }
                >
                  Test Email
                </Button>
                <Chip
                  label={
                    smtpHost && smtpFrom && smtpTo ? 'Configured' : 'Incomplete'
                  }
                  size="small"
                  color={smtpHost && smtpFrom && smtpTo ? 'success' : 'warning'}
                />
              </CardActions>
            )}
          </Card>
        </Grid>

        {/* Discord Notifications */}
        <Grid item xs={12} md={6}>
          <Card sx={{ height: '100%' }}>
            <CardContent>
              <Box sx={{ display: 'flex', alignItems: 'center', mb: 2 }}>
                <DiscordIcon sx={{ mr: 1, color: 'primary.main' }} />
                <Typography variant="h6" color="primary">
                  Discord
                </Typography>
                <Box sx={{ flexGrow: 1 }} />
                <Switch
                  checked={discordEnabled}
                  onChange={e => setDiscordEnabled(e.target.checked)}
                  disabled={!backendAvailable}
                />
              </Box>

              <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>
                Send notifications to Discord channels via webhooks.
              </Typography>

              {discordEnabled && (
                <Box sx={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
                  <TextField
                    fullWidth
                    label="Webhook URL"
                    value={discordWebhook}
                    onChange={e => setDiscordWebhook(e.target.value)}
                    placeholder="https://discord.com/api/webhooks/..."
                    size="small"
                    disabled={!backendAvailable}
                  />

                  <TextField
                    fullWidth
                    label="Bot Username"
                    value={discordUsername}
                    onChange={e => setDiscordUsername(e.target.value)}
                    size="small"
                    disabled={!backendAvailable}
                  />

                  <TextField
                    fullWidth
                    label="Avatar URL"
                    value={discordAvatar}
                    onChange={e => setDiscordAvatar(e.target.value)}
                    placeholder="https://example.com/avatar.png (optional)"
                    size="small"
                    disabled={!backendAvailable}
                  />
                </Box>
              )}
            </CardContent>

            {discordEnabled && (
              <CardActions>
                <Button
                  size="small"
                  startIcon={<TestIcon />}
                  onClick={() => testNotification('discord')}
                  disabled={!backendAvailable || !discordWebhook}
                >
                  Test Discord
                </Button>
                <Chip
                  label={discordWebhook ? 'Configured' : 'Incomplete'}
                  size="small"
                  color={discordWebhook ? 'success' : 'warning'}
                />
              </CardActions>
            )}
          </Card>
        </Grid>

        {/* Telegram Notifications */}
        <Grid item xs={12} md={6}>
          <Card sx={{ height: '100%' }}>
            <CardContent>
              <Box sx={{ display: 'flex', alignItems: 'center', mb: 2 }}>
                <TelegramIcon sx={{ mr: 1, color: 'primary.main' }} />
                <Typography variant="h6" color="primary">
                  Telegram
                </Typography>
                <Box sx={{ flexGrow: 1 }} />
                <Switch
                  checked={telegramEnabled}
                  onChange={e => setTelegramEnabled(e.target.checked)}
                  disabled={!backendAvailable}
                />
              </Box>

              <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>
                Send notifications via Telegram bot messages.
              </Typography>

              {telegramEnabled && (
                <Box sx={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
                  <TextField
                    fullWidth
                    label="Bot Token"
                    value={telegramToken}
                    onChange={e => setTelegramToken(e.target.value)}
                    placeholder="123456:ABCDEF"
                    size="small"
                    disabled={!backendAvailable}
                  />

                  <TextField
                    fullWidth
                    label="Chat ID"
                    value={telegramChatId}
                    onChange={e => setTelegramChatId(e.target.value)}
                    placeholder="12345678"
                    size="small"
                    disabled={!backendAvailable}
                  />
                </Box>
              )}
            </CardContent>

            {telegramEnabled && (
              <CardActions>
                <Button
                  size="small"
                  startIcon={<TestIcon />}
                  onClick={() => testNotification('telegram')}
                  disabled={
                    !backendAvailable || !telegramToken || !telegramChatId
                  }
                >
                  Test Telegram
                </Button>
                <Chip
                  label={
                    telegramToken && telegramChatId
                      ? 'Configured'
                      : 'Incomplete'
                  }
                  size="small"
                  color={
                    telegramToken && telegramChatId ? 'success' : 'warning'
                  }
                />
              </CardActions>
            )}
          </Card>
        </Grid>

        {/* Push Notifications */}
        <Grid item xs={12} md={6}>
          <Card sx={{ height: '100%' }}>
            <CardContent>
              <Box sx={{ display: 'flex', alignItems: 'center', mb: 2 }}>
                <PushIcon sx={{ mr: 1, color: 'primary.main' }} />
                <Typography variant="h6" color="primary">
                  Push (Pushover)
                </Typography>
                <Box sx={{ flexGrow: 1 }} />
                <Switch
                  checked={pushEnabled}
                  onChange={e => setPushEnabled(e.target.checked)}
                  disabled={!backendAvailable}
                />
              </Box>

              <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>
                Send push notifications using the Pushover service.
              </Typography>

              {pushEnabled && (
                <Box sx={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
                  <TextField
                    fullWidth
                    label="API Token"
                    value={pushoverToken}
                    onChange={e => setPushoverToken(e.target.value)}
                    size="small"
                    disabled={!backendAvailable}
                  />

                  <TextField
                    fullWidth
                    label="User Key"
                    value={pushoverUser}
                    onChange={e => setPushoverUser(e.target.value)}
                    size="small"
                    disabled={!backendAvailable}
                  />
                </Box>
              )}
            </CardContent>

            {pushEnabled && (
              <CardActions>
                <Button
                  size="small"
                  startIcon={<TestIcon />}
                  onClick={() => testNotification('push')}
                  disabled={
                    !backendAvailable || !pushoverToken || !pushoverUser
                  }
                >
                  Test Push
                </Button>
                <Chip
                  label={
                    pushoverToken && pushoverUser ? 'Configured' : 'Incomplete'
                  }
                  size="small"
                  color={pushoverToken && pushoverUser ? 'success' : 'warning'}
                />
              </CardActions>
            )}
          </Card>
        </Grid>

        {/* Webhook Notifications */}
        <Grid item xs={12} md={6}>
          <Card sx={{ height: '100%' }}>
            <CardContent>
              <Box sx={{ display: 'flex', alignItems: 'center', mb: 2 }}>
                <WebhookIcon sx={{ mr: 1, color: 'primary.main' }} />
                <Typography variant="h6" color="primary">
                  Webhook
                </Typography>
                <Box sx={{ flexGrow: 1 }} />
                <Switch
                  checked={webhookEnabled}
                  onChange={e => setWebhookEnabled(e.target.checked)}
                  disabled={!backendAvailable}
                />
              </Box>

              <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>
                Trigger a generic webhook on events.
              </Typography>

              {webhookEnabled && (
                <Box sx={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
                  <TextField
                    fullWidth
                    label="URL"
                    value={webhookUrl}
                    onChange={e => setWebhookUrl(e.target.value)}
                    placeholder="https://example.com/webhook"
                    size="small"
                    disabled={!backendAvailable}
                  />

                  <FormControl
                    fullWidth
                    size="small"
                    disabled={!backendAvailable}
                  >
                    <InputLabel>Method</InputLabel>
                    <Select
                      label="Method"
                      value={webhookMethod}
                      onChange={e => setWebhookMethod(e.target.value)}
                    >
                      <MenuItem value="POST">POST</MenuItem>
                      <MenuItem value="PUT">PUT</MenuItem>
                      <MenuItem value="GET">GET</MenuItem>
                    </Select>
                  </FormControl>

                  <TextField
                    fullWidth
                    label="Headers (JSON)"
                    value={webhookHeaders}
                    onChange={e => setWebhookHeaders(e.target.value)}
                    placeholder='{"Authorization":"token"}'
                    multiline
                    minRows={2}
                    size="small"
                    disabled={!backendAvailable}
                  />
                </Box>
              )}
            </CardContent>

            {webhookEnabled && (
              <CardActions>
                <Button
                  size="small"
                  startIcon={<TestIcon />}
                  onClick={() => testNotification('webhook')}
                  disabled={!backendAvailable || !webhookUrl}
                >
                  Test Webhook
                </Button>
                <Chip
                  label={webhookUrl ? 'Configured' : 'Incomplete'}
                  size="small"
                  color={webhookUrl ? 'success' : 'warning'}
                />
              </CardActions>
            )}
          </Card>
        </Grid>
      </Grid>

      <Box sx={{ mt: 3, display: 'flex', justifyContent: 'flex-end' }}>
        <Button
          variant="contained"
          onClick={handleSave}
          disabled={!backendAvailable}
          size="large"
        >
          Save Notification Settings
        </Button>
      </Box>

      {/* Test dialog */}
      <Dialog
        open={testDialog.open}
        onClose={() =>
          !testDialog.loading &&
          setTestDialog({ open: false, type: '', loading: false })
        }
      >
        <DialogTitle>Testing {testDialog.type} Notification</DialogTitle>
        <DialogContent>
          {testDialog.loading ? (
            <Box sx={{ display: 'flex', alignItems: 'center', gap: 2, py: 2 }}>
              <CircularProgress size={24} />
              <Typography>Sending test notification...</Typography>
            </Box>
          ) : (
            <Typography>Test notification sent!</Typography>
          )}
        </DialogContent>
        {!testDialog.loading && (
          <DialogActions>
            <Button
              onClick={() =>
                setTestDialog({ open: false, type: '', loading: false })
              }
            >
              Close
            </Button>
          </DialogActions>
        )}
      </Dialog>

      {/* Reset dialog */}
      <Dialog open={resetDialog} onClose={() => setResetDialog(false)}>
        <DialogTitle>Reset All Notification Settings</DialogTitle>
        <DialogContent>
          <Typography>
            This will reset all notification settings to their default values.
            This action cannot be undone.
          </Typography>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setResetDialog(false)}>Cancel</Button>
          <Button
            onClick={resetAllNotifications}
            color="error"
            variant="contained"
          >
            Reset All
          </Button>
        </DialogActions>
      </Dialog>
    </Box>
  );
}

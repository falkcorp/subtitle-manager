// file: webui/src/components/AuthSettings.jsx
import {
  Box,
  Button,
  Card,
  CardContent,
  CardActions,
  FormControlLabel,
  Grid,
  Switch,
  TextField,
  Typography,
  Alert,
  IconButton,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
} from '@mui/material';
import {
  GitHub as GitHubIcon,
  Key as ApiKeyIcon,
  Password as PasswordIcon,
  Security as OAuthIcon,
  Refresh as RefreshIcon,
  Add as AddIcon,
  Delete as DeleteIcon,
  Visibility as ShowIcon,
  VisibilityOff as HideIcon,
} from '@mui/icons-material';
import { useEffect, useState } from 'react';
import { apiFetch } from '../services/api.js';

/**
 * Enhanced AuthSettings with card-based UI for each authentication method.
 *
 * @param {Object} props - Component properties.
 * @param {Object} props.config - Current configuration values.
 * @param {Function} props.onSave - Callback invoked with updated settings.
 * @param {boolean} [props.backendAvailable=true] - Whether backend APIs are reachable.
 * @returns {JSX.Element} AuthSettings component.
 */
export default function AuthSettings({
  config,
  onSave,
  backendAvailable = true,
}) {
  // Password Authentication
  const [passwordAuthEnabled, setPasswordAuthEnabled] = useState(true);
  const [requireStrongPasswords, setRequireStrongPasswords] = useState(true);
  const [passwordExpiry, setPasswordExpiry] = useState(90);

  // API Key Authentication
  const [apiKeyEnabled, setApiKeyEnabled] = useState(true);
  const [apiKeyExpiry, setApiKeyExpiry] = useState(0); // 0 = no expiry

  // GitHub OAuth
  const [githubEnabled, setGithubEnabled] = useState(false);
  const [githubClientId, setGithubClientId] = useState('');
  const [githubClientSecret, setGithubClientSecret] = useState('');
  const [githubRedirectUrl, setGithubRedirectUrl] = useState('');
  const [showGithubSecret, setShowGithubSecret] = useState(false);

  // Generic OAuth2
  const [genericOAuthEnabled, setGenericOAuthEnabled] = useState(false);
  const [oauthProvider, setOauthProvider] = useState('');
  const [oauthClientId, setOauthClientId] = useState('');
  const [oauthClientSecret, setOauthClientSecret] = useState('');
  const [oauthAuthUrl, setOauthAuthUrl] = useState('');
  const [oauthTokenUrl, setOauthTokenUrl] = useState('');
  const [oauthUserUrl, setOauthUserUrl] = useState('');
  const [showOAuthSecret, setShowOAuthSecret] = useState(false);

  // Dialog states
  const [resetGithubDialog, setResetGithubDialog] = useState(false);
  const [resetOAuthDialog, setResetOAuthDialog] = useState(false);

  useEffect(() => {
    if (config) {
      // Password settings
      setPasswordAuthEnabled(config.password_auth_enabled !== false);
      setRequireStrongPasswords(config.require_strong_passwords !== false);
      setPasswordExpiry(config.password_expiry || 90);

      // API Key settings
      setApiKeyEnabled(config.api_key_enabled !== false);
      setApiKeyExpiry(config.api_key_expiry || 0);

      // GitHub OAuth settings
      setGithubEnabled(config.github_oauth_enabled || false);
      setGithubClientId(config.github_client_id || '');
      setGithubClientSecret(config.github_client_secret || '');
      setGithubRedirectUrl(
        config.github_redirect_url ||
          `${window.location.origin}/api/oauth/github/callback`
      );

      // Generic OAuth settings
      setGenericOAuthEnabled(config.generic_oauth_enabled || false);
      setOauthProvider(config.oauth_provider || '');
      setOauthClientId(config.oauth_client_id || '');
      setOauthClientSecret(config.oauth_client_secret || '');
      setOauthAuthUrl(config.oauth_auth_url || '');
      setOauthTokenUrl(config.oauth_token_url || '');
      setOauthUserUrl(config.oauth_user_url || '');
    }
  }, [config]);

  const handleSave = () => {
    const newConfig = {
      // Password authentication
      password_auth_enabled: passwordAuthEnabled,
      require_strong_passwords: requireStrongPasswords,
      password_expiry: parseInt(passwordExpiry, 10),

      // API Key authentication
      api_key_enabled: apiKeyEnabled,
      api_key_expiry: parseInt(apiKeyExpiry, 10),

      // GitHub OAuth
      github_oauth_enabled: githubEnabled,
      github_client_id: githubClientId,
      github_client_secret: githubClientSecret,
      github_redirect_url: githubRedirectUrl,

      // Generic OAuth
      generic_oauth_enabled: genericOAuthEnabled,
      oauth_provider: oauthProvider,
      oauth_client_id: oauthClientId,
      oauth_client_secret: oauthClientSecret,
      oauth_auth_url: oauthAuthUrl,
      oauth_token_url: oauthTokenUrl,
      oauth_user_url: oauthUserUrl,
    };

    onSave(newConfig);
  };

  const generateGithubCredentials = async () => {
    try {
      const response = await apiFetch('/api/oauth/github/generate', {
        method: 'POST',
      });
      if (response.ok) {
        const data = await response.json();
        setGithubClientId(data.client_id);
        setGithubClientSecret(data.client_secret);
        alert('New GitHub OAuth credentials generated successfully');
      }
    } catch (error) {
      alert('Failed to generate credentials: ' + error.message);
    }
  };

  const regenerateGithubSecret = async () => {
    try {
      const response = await apiFetch('/api/oauth/github/regenerate', {
        method: 'POST',
      });
      if (response.ok) {
        const data = await response.json();
        setGithubClientSecret(data.client_secret);
        alert('GitHub client secret regenerated successfully');
      }
    } catch (error) {
      alert('Failed to regenerate secret: ' + error.message);
    }
  };

  const resetGithubConfig = () => {
    setGithubClientId('');
    setGithubClientSecret('');
    setGithubRedirectUrl(`${window.location.origin}/api/oauth/github/callback`);
    setGithubEnabled(false);
    setResetGithubDialog(false);
    alert('GitHub OAuth configuration reset to defaults');
  };

  const resetOAuthConfig = () => {
    setOauthProvider('');
    setOauthClientId('');
    setOauthClientSecret('');
    setOauthAuthUrl('');
    setOauthTokenUrl('');
    setOauthUserUrl('');
    setGenericOAuthEnabled(false);
    setResetOAuthDialog(false);
    alert('Generic OAuth configuration reset to defaults');
  };

  return (
    <Box sx={{ maxWidth: 1200 }}>
      <Typography variant="h6" gutterBottom>
        Authentication Settings
      </Typography>

      {!backendAvailable && (
        <Alert severity="warning" sx={{ mb: 3 }}>
          Backend service is not available. Authentication settings cannot be
          modified.
        </Alert>
      )}

      <Grid container spacing={3}>
        {/* Password Authentication Card */}
        <Grid item xs={12} md={6}>
          <Card sx={{ height: '100%' }}>
            <CardContent>
              <Box sx={{ display: 'flex', alignItems: 'center', mb: 2 }}>
                <PasswordIcon sx={{ mr: 1, color: 'primary.main' }} />
                <Typography variant="h6" color="primary">
                  Password Authentication
                </Typography>
                <Box sx={{ flexGrow: 1 }} />
                <Switch
                  checked={passwordAuthEnabled}
                  onChange={e => setPasswordAuthEnabled(e.target.checked)}
                  disabled={!backendAvailable}
                />
              </Box>

              <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>
                Traditional username and password authentication for local
                users.
              </Typography>

              {passwordAuthEnabled && (
                <Box sx={{ mt: 2 }}>
                  <FormControlLabel
                    control={
                      <Switch
                        checked={requireStrongPasswords}
                        onChange={e =>
                          setRequireStrongPasswords(e.target.checked)
                        }
                        disabled={!backendAvailable}
                      />
                    }
                    label="Require Strong Passwords"
                    sx={{ mb: 1 }}
                  />

                  <TextField
                    fullWidth
                    label="Password Expiry (days)"
                    type="number"
                    value={passwordExpiry}
                    onChange={e => setPasswordExpiry(e.target.value)}
                    helperText="Set to 0 for no expiry"
                    disabled={!backendAvailable}
                    sx={{ mt: 1 }}
                  />
                </Box>
              )}
            </CardContent>
          </Card>
        </Grid>

        {/* API Key Authentication Card */}
        <Grid item xs={12} md={6}>
          <Card sx={{ height: '100%' }}>
            <CardContent>
              <Box sx={{ display: 'flex', alignItems: 'center', mb: 2 }}>
                <ApiKeyIcon sx={{ mr: 1, color: 'primary.main' }} />
                <Typography variant="h6" color="primary">
                  API Key Authentication
                </Typography>
                <Box sx={{ flexGrow: 1 }} />
                <Switch
                  checked={apiKeyEnabled}
                  onChange={e => setApiKeyEnabled(e.target.checked)}
                  disabled={!backendAvailable}
                />
              </Box>

              <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>
                API key-based authentication for automated access and
                integrations.
              </Typography>

              {apiKeyEnabled && (
                <TextField
                  fullWidth
                  label="API Key Expiry (days)"
                  type="number"
                  value={apiKeyExpiry}
                  onChange={e => setApiKeyExpiry(e.target.value)}
                  helperText="Set to 0 for no expiry"
                  disabled={!backendAvailable}
                  sx={{ mt: 2 }}
                />
              )}
            </CardContent>
          </Card>
        </Grid>

        {/* GitHub OAuth Card */}
        <Grid item xs={12} md={6}>
          <Card sx={{ height: '100%' }}>
            <CardContent>
              <Box sx={{ display: 'flex', alignItems: 'center', mb: 2 }}>
                <GitHubIcon sx={{ mr: 1, color: 'primary.main' }} />
                <Typography variant="h6" color="primary">
                  GitHub OAuth
                </Typography>
                <Box sx={{ flexGrow: 1 }} />
                <Switch
                  checked={githubEnabled}
                  onChange={e => setGithubEnabled(e.target.checked)}
                  disabled={!backendAvailable}
                />
              </Box>

              <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>
                Allow users to authenticate using their GitHub accounts.
              </Typography>

              {githubEnabled && (
                <Box
                  sx={{
                    mt: 2,
                    display: 'flex',
                    flexDirection: 'column',
                    gap: 2,
                  }}
                >
                  <TextField
                    fullWidth
                    label="Client ID"
                    value={githubClientId}
                    onChange={e => setGithubClientId(e.target.value)}
                    disabled={!backendAvailable}
                  />

                  <TextField
                    fullWidth
                    label="Client Secret"
                    type={showGithubSecret ? 'text' : 'password'}
                    value={githubClientSecret}
                    onChange={e => setGithubClientSecret(e.target.value)}
                    disabled={!backendAvailable}
                    InputProps={{
                      endAdornment: (
                        <IconButton
                          onClick={() => setShowGithubSecret(!showGithubSecret)}
                          edge="end"
                        >
                          {showGithubSecret ? <HideIcon /> : <ShowIcon />}
                        </IconButton>
                      ),
                    }}
                  />

                  <TextField
                    fullWidth
                    label="Redirect URL"
                    value={githubRedirectUrl}
                    onChange={e => setGithubRedirectUrl(e.target.value)}
                    disabled={!backendAvailable}
                    helperText="Copy this URL to your GitHub OAuth app settings"
                  />
                </Box>
              )}
            </CardContent>

            {githubEnabled && (
              <CardActions>
                <Button
                  size="small"
                  startIcon={<AddIcon />}
                  onClick={generateGithubCredentials}
                  disabled={!backendAvailable}
                >
                  Generate New
                </Button>
                <Button
                  size="small"
                  startIcon={<RefreshIcon />}
                  onClick={regenerateGithubSecret}
                  disabled={!backendAvailable}
                >
                  Regenerate Secret
                </Button>
                <Button
                  size="small"
                  startIcon={<DeleteIcon />}
                  onClick={() => setResetGithubDialog(true)}
                  disabled={!backendAvailable}
                  color="error"
                >
                  Reset
                </Button>
              </CardActions>
            )}
          </Card>
        </Grid>

        {/* Generic OAuth Card */}
        <Grid item xs={12} md={6}>
          <Card sx={{ height: '100%' }}>
            <CardContent>
              <Box sx={{ display: 'flex', alignItems: 'center', mb: 2 }}>
                <OAuthIcon sx={{ mr: 1, color: 'primary.main' }} />
                <Typography variant="h6" color="primary">
                  Generic OAuth2
                </Typography>
                <Box sx={{ flexGrow: 1 }} />
                <Switch
                  checked={genericOAuthEnabled}
                  onChange={e => setGenericOAuthEnabled(e.target.checked)}
                  disabled={!backendAvailable}
                />
              </Box>

              <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>
                Configure authentication with any OAuth2-compatible provider.
              </Typography>

              {genericOAuthEnabled && (
                <Box
                  sx={{
                    mt: 2,
                    display: 'flex',
                    flexDirection: 'column',
                    gap: 2,
                  }}
                >
                  <TextField
                    fullWidth
                    label="Provider Name"
                    value={oauthProvider}
                    onChange={e => setOauthProvider(e.target.value)}
                    placeholder="e.g., Google, Microsoft, etc."
                    disabled={!backendAvailable}
                  />

                  <TextField
                    fullWidth
                    label="Client ID"
                    value={oauthClientId}
                    onChange={e => setOauthClientId(e.target.value)}
                    disabled={!backendAvailable}
                  />

                  <TextField
                    fullWidth
                    label="Client Secret"
                    type={showOAuthSecret ? 'text' : 'password'}
                    value={oauthClientSecret}
                    onChange={e => setOauthClientSecret(e.target.value)}
                    disabled={!backendAvailable}
                    InputProps={{
                      endAdornment: (
                        <IconButton
                          onClick={() => setShowOAuthSecret(!showOAuthSecret)}
                          edge="end"
                        >
                          {showOAuthSecret ? <HideIcon /> : <ShowIcon />}
                        </IconButton>
                      ),
                    }}
                  />

                  <TextField
                    fullWidth
                    label="Authorization URL"
                    value={oauthAuthUrl}
                    onChange={e => setOauthAuthUrl(e.target.value)}
                    placeholder="https://provider.com/oauth/authorize"
                    disabled={!backendAvailable}
                  />

                  <TextField
                    fullWidth
                    label="Token URL"
                    value={oauthTokenUrl}
                    onChange={e => setOauthTokenUrl(e.target.value)}
                    placeholder="https://provider.com/oauth/token"
                    disabled={!backendAvailable}
                  />

                  <TextField
                    fullWidth
                    label="User Info URL"
                    value={oauthUserUrl}
                    onChange={e => setOauthUserUrl(e.target.value)}
                    placeholder="https://provider.com/oauth/userinfo"
                    disabled={!backendAvailable}
                  />
                </Box>
              )}
            </CardContent>

            {genericOAuthEnabled && (
              <CardActions>
                <Button
                  size="small"
                  startIcon={<DeleteIcon />}
                  onClick={() => setResetOAuthDialog(true)}
                  disabled={!backendAvailable}
                  color="error"
                >
                  Reset Configuration
                </Button>
              </CardActions>
            )}
          </Card>
        </Grid>
      </Grid>

      {/* Save Button */}
      <Box sx={{ mt: 3, display: 'flex', justifyContent: 'flex-end' }}>
        <Button
          variant="contained"
          onClick={handleSave}
          disabled={!backendAvailable}
          size="large"
        >
          Save Authentication Settings
        </Button>
      </Box>

      {/* Reset Dialogs */}
      <Dialog
        open={resetGithubDialog}
        onClose={() => setResetGithubDialog(false)}
      >
        <DialogTitle>Reset GitHub OAuth Configuration</DialogTitle>
        <DialogContent>
          <Typography>
            This will reset all GitHub OAuth settings to their default values.
            This action cannot be undone.
          </Typography>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setResetGithubDialog(false)}>Cancel</Button>
          <Button onClick={resetGithubConfig} color="error" variant="contained">
            Reset
          </Button>
        </DialogActions>
      </Dialog>

      <Dialog
        open={resetOAuthDialog}
        onClose={() => setResetOAuthDialog(false)}
      >
        <DialogTitle>Reset Generic OAuth Configuration</DialogTitle>
        <DialogContent>
          <Typography>
            This will reset all generic OAuth settings to their default values.
            This action cannot be undone.
          </Typography>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setResetOAuthDialog(false)}>Cancel</Button>
          <Button onClick={resetOAuthConfig} color="error" variant="contained">
            Reset
          </Button>
        </DialogActions>
      </Dialog>
    </Box>
  );
}

// file: webui/src/Settings.jsx
// version: 1.2.0
// guid: b1c2d3e4-f5a6-4b7c-8d9e-0a1b2c3d4e5f
// last-edited: 2026-08-12

import {
  Security as AuthIcon,
  Storage as DatabaseIcon,
  Settings as GeneralIcon,
  CloudUpload as ImportIcon,
  Language as LanguageIcon,
  Notifications as NotificationIcon,
  CloudDownload as ProvidersIcon,
  Refresh as RefreshIcon,
  Label as TagsIcon,
  People as UsersIcon,
  Webhook as WebhookIcon,
} from '@mui/icons-material';
import {
  Alert,
  Box,
  Button,
  CircularProgress,
  Grid,
  Paper,
  Snackbar,
  Tab,
  Tabs,
  Typography,
} from '@mui/material';
import { lazy, Suspense, useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { useParams } from 'react-router-dom';
import ImportDialog from './components/ImportDialog.jsx';
import LanguageSelector from './components/LanguageSelector.jsx';
import LoadingComponent from './components/LoadingComponent.jsx';
import ProviderCard from './components/ProviderCard.jsx';
import ProviderConfigDialog from './components/ProviderConfigDialog.jsx';
import BackButton from './components/BackButton.jsx';
import { apiService } from './services/api.js';
import { apiFetch } from './services/api.js';

// Lazy load settings components for better performance
const AuthSettings = lazy(() => import('./components/AuthSettings.jsx'));
const DatabaseSettings = lazy(
  () => import('./components/DatabaseSettings.jsx')
);
const GeneralSettings = lazy(() => import('./components/GeneralSettings.jsx'));
const LanguageProfiles = lazy(
  () => import('./components/LanguageProfiles.jsx')
);
const NotificationSettings = lazy(
  () => import('./components/NotificationSettings.jsx')
);
const WebhookSettings = lazy(() => import('./components/WebhookSettings.jsx'));
const UserManagement = lazy(() => import('./UserManagement.jsx'));
const TagManagement = lazy(() => import('./TagManagement.jsx'));

/**
 * Settings component with modern tabbed interface for managing all aspects
 * of subtitle manager configuration. Includes provider management similar to Bazarr.
 * @param {Object} props - Component props
 * @param {boolean} props.backendAvailable - Whether the backend service is available
 */
export default function Settings({ backendAvailable = true }) {
  const { t } = useTranslation();
  const { section } = useParams();
  const [activeTab, setActiveTab] = useState(0);
  const [_config, setConfig] = useState(null);
  const [providers, setProviders] = useState([]);
  const [status, setStatus] = useState('');
  const [snackbarOpen, setSnackbarOpen] = useState(false);
  const [importDialogOpen, setImportDialogOpen] = useState(false);
  const [providerConfigDialog, setProviderConfigDialog] = useState({
    open: false,
    provider: null,
  });
  const [_bazarrConfig] = useState(null);
  const [bazarrURL, setBazarrURL] = useState('');
  const [bazarrAPIKey, setBazarrAPIKey] = useState('');
  const [bazarrMappings, setBazarrMappings] = useState([]);
  const [selectedSettings, setSelectedSettings] = useState({});
  const [previewConfig, setPreviewConfig] = useState(null);
  const [importLoading, setImportLoading] = useState(false);
  const [importError, setImportError] = useState('');
  const [importing, setImporting] = useState(false);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState(null);

  useEffect(() => {
    if (backendAvailable) {
      loadConfig();
      loadProviders();
    }
  }, [backendAvailable]); // eslint-disable-line react-hooks/exhaustive-deps

  /**
   * Load current configuration from the server
   */
  const loadConfig = async () => {
    if (!backendAvailable) return;

    setLoading(true);
    try {
      setError(null);
      const response = await apiService.get('/api/config');
      if (response.ok) {
        const data = await response.json();
        setConfig(data);
      } else if (response.status === 403) {
        setError('Permission denied');
      } else {
        setError('Failed to load configuration');
      }
    } catch (error) {
      console.error('Failed to load configuration:', error);
      setStatus('Failed to load configuration');
      setError('Failed to load configuration');
    } finally {
      setLoading(false);
    }
  };

  /**
   * Load available providers and their current configuration
   */
  const loadProviders = async () => {
    if (!backendAvailable) return;

    try {
      setError(null);
      const response = await apiService.get('/api/providers');
      if (response.ok) {
        const data = await response.json();
        // Transform provider data to include display information
        const providersWithMetadata = data.map(provider => ({
          ...provider,
          displayName: formatProviderName(provider.name),
          description: getProviderDescription(provider.name),
          languages: getProviderLanguages(provider.name),
          configured: hasRequiredConfig(provider),
        }));
        setProviders(providersWithMetadata);
      }
    } catch (error) {
      console.error('Failed to load providers:', error);
      setError('Failed to load providers');
    }
  };

  /**
   * Format provider name for display (e.g., "opensubtitles" -> "OpenSubtitles")
   */
  const formatProviderName = name => {
    const specialNames = {
      opensubtitles: 'OpenSubtitles',
      opensubtitlescom: 'OpenSubtitles.com',
      opensubtitlesvip: 'OpenSubtitles VIP',
      addic7ed: 'Addic7ed',
      podnapisi: 'Podnapisi.NET',
      subscene: 'Subscene',
      yifysubtitles: 'YIFY Subtitles',
      turkcealtyazi: 'Türkçe Altyazı',
      greeksubtitles: 'Greek Subtitles',
      legendasdivx: 'Legendas DivX',
      legendasnet: 'Legendas.NET',
      napiprojekt: 'NapiProjekt',
    };

    return (
      specialNames[name] ||
      name
        .split(/(?=[A-Z])/)
        .map(word => word.charAt(0).toUpperCase() + word.slice(1))
        .join(' ')
    );
  };

  /**
   * Get provider description based on provider type
   */
  const getProviderDescription = name => {
    const descriptions = {
      opensubtitles: 'Large community-driven subtitle database',
      addic7ed: 'TV shows subtitle provider with high quality subs',
      subscene: 'Popular subtitle site for movies and TV shows',
      whisper: 'AI-powered speech recognition for subtitle generation',
      embedded: 'Extract subtitles embedded in media files',
      generic: 'Generic HTTP/API-based subtitle provider',
    };

    return (
      descriptions[name] || `${formatProviderName(name)} subtitle provider`
    );
  };

  /**
   * Get commonly supported languages for a provider
   */
  const getProviderLanguages = name => {
    // This would ideally come from the provider metadata
    const commonLanguages = ['en', 'es', 'fr', 'de', 'it', 'pt'];
    const specialLanguages = {
      turkcealtyazi: ['tr', 'en'],
      greeksubtitles: ['el', 'en'],
      napiprojekt: ['pl', 'en'],
      legendasdivx: ['pt', 'es', 'en'],
    };

    return specialLanguages[name] || commonLanguages;
  };

  /**
   * Check if provider has required configuration
   */
  const hasRequiredConfig = provider => {
    if (!provider.config) return false;

    const requiredFields = {
      opensubtitles: ['apiKey'],
      addic7ed: ['username', 'password'],
      generic: ['baseUrl'],
    };

    const required = requiredFields[provider.name] || [];
    return required.every(field => provider.config[field]);
  };

  /**
   * Toggle provider enabled state
   */
  // Provider settings are written through POST /api/config as flat dotted
  // keys, which is the shape pkg/providers/config.go reads
  // (providers.<name>.enabled / .priority / .tags).
  //
  // This used to PATCH /api/providers/{name}, which is not a mounted route —
  // it fell into the /api/providers/ subtree handler and 404'd. Combined with
  // an `if (response.ok)` that had no `else`, every toggle reported success
  // and changed nothing.
  const handleProviderToggle = async (providerName, enabled) => {
    try {
      const response = await apiFetch('/api/config', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          [`providers.${providerName}.enabled`]: enabled,
        }),
      });

      if (!response.ok) {
        // Never leave the switch showing a state the server did not accept.
        setStatus(
          `Failed to ${enabled ? 'enable' : 'disable'} ${formatProviderName(providerName)} (${response.status})`
        );
        setSnackbarOpen(true);
        return;
      }

      setProviders(prev =>
        prev.map(p => (p.name === providerName ? { ...p, enabled } : p))
      );
      setStatus(
        `${formatProviderName(providerName)} ${enabled ? 'enabled' : 'disabled'}`
      );
      setSnackbarOpen(true);
    } catch (error) {
      setStatus(
        `Failed to update ${formatProviderName(providerName)}: ${error.message}`
      );
      setSnackbarOpen(true);
    }
  };

  /**
   * Open provider configuration dialog
   */
  const handleProviderConfigure = provider => {
    setProviderConfigDialog({ open: true, provider });
  };

  /**
   * Save provider configuration
   */
  // Returns true when the configuration was actually persisted, so the dialog
  // can stay open on failure instead of closing as though it had saved.
  //
  // Like the toggle above, this used to POST /api/providers/{name}/config —
  // an unmounted route swallowed by the /api/providers/ subtree.
  const handleProviderSave = async provider => {
    // Flatten to the dotted keys the server reads. Nested objects posted to
    // /api/config are stored verbatim under the literal key, which is how
    // settings pages here have silently saved data nothing ever reads.
    const payload = Object.fromEntries(
      Object.entries(provider.config ?? {}).map(([field, value]) => [
        `providers.${provider.name}.${field}`,
        value,
      ])
    );
    payload[`providers.${provider.name}.enabled`] = provider.enabled !== false;

    try {
      const response = await apiFetch('/api/config', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(payload),
      });

      if (!response.ok) {
        setStatus(
          `Failed to save ${formatProviderName(provider.name)} configuration (${response.status})`
        );
        setSnackbarOpen(true);
        return false;
      }

      setProviders(prev =>
        prev.map(p =>
          p.name === provider.name
            ? { ...provider, configured: hasRequiredConfig(provider) }
            : p
        )
      );
      setProviderConfigDialog({ open: false, provider: null });
      setStatus(`${formatProviderName(provider.name)} configuration saved`);
      setSnackbarOpen(true);
      return true;
    } catch (error) {
      setStatus(
        `Failed to save ${formatProviderName(provider.name)} configuration: ${error.message}`
      );
      setSnackbarOpen(true);
      return false;
    }
  };

  /**
   * Save general configuration values via the API
   *
   * @param {Object} values - Key/value pairs to persist
   */
  const saveSettings = async values => {
    try {
      const response = await apiFetch('/api/config', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(values),
      });
      if (response.ok) {
        setStatus('Settings saved');
        setSnackbarOpen(true);
        await loadConfig();
      } else {
        setStatus('Failed to save settings');
        setSnackbarOpen(true);
      }
    } catch {
      setError('Failed to update settings');
    }
  };

  /**
   * Import settings from Bazarr
   */
  const openImportDialog = () => {
    setBazarrURL('');
    setBazarrAPIKey('');
    setBazarrMappings([]);
    setSelectedSettings({});
    setPreviewConfig(null);
    setImportError('');
    setImportDialogOpen(true);
  };

  const previewBazarr = async () => {
    if (!bazarrURL || !bazarrAPIKey) {
      setImportError('Please provide URL and API key');
      return;
    }
    setImportLoading(true);
    setImportError('');
    try {
      const res = await apiFetch('/api/bazarr/preview', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ url: bazarrURL, api_key: bazarrAPIKey }),
      });
      if (res.ok) {
        const data = await res.json();
        setPreviewConfig(data.preview);
        setBazarrMappings(data.mappings || []);
        const selected = {};
        (data.mappings || []).forEach(m => {
          selected[m.key] = true;
        });
        setSelectedSettings(selected);
      } else {
        const text = await res.text();
        setImportError(text || 'Failed to connect');
      }
    } catch (err) {
      setImportError('Network error: ' + err.message);
    } finally {
      setImportLoading(false);
    }
  };

  const importFromBazarr = async () => {
    setImporting(true);
    try {
      const keys = Object.keys(selectedSettings).filter(
        k => selectedSettings[k]
      );
      const response = await apiFetch('/api/bazarr/import', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          url: bazarrURL,
          api_key: bazarrAPIKey,
          keys,
        }),
      });
      if (response.ok) {
        setStatus('Settings imported from Bazarr successfully');
        setImportDialogOpen(false);
        setSnackbarOpen(true);
        await loadConfig();
        await loadProviders();
      } else {
        setStatus('Failed to import from Bazarr');
      }
    } catch {
      setStatus('Error importing from Bazarr');
    } finally {
      setImporting(false);
    }
  };

  /**
   * Render provider management tab
   */
  const renderProvidersTab = () => (
    <Box>
      <Box
        display="flex"
        justifyContent="space-between"
        alignItems="center"
        mb={3}
      >
        <Typography variant="h5">Subtitle Providers</Typography>
        <Button
          variant="outlined"
          startIcon={
            importing ? <RefreshIcon className="spin" /> : <ImportIcon />
          }
          onClick={openImportDialog}
          disabled={importing}
        >
          Import from Bazarr
        </Button>
      </Box>

      <Alert severity="info" sx={{ mb: 3 }}>
        Enable and configure subtitle providers. Providers are checked in order,
        so arrange them by preference. Disabled providers are skipped during
        searches.
      </Alert>

      <Grid container spacing={3}>
        {providers
          .filter(p => p.enabled)
          .map(provider => (
            <Grid size={{ xs: 12, sm: 6, md: 4, lg: 3 }} key={provider.name}>
              <ProviderCard
                provider={provider}
                onToggle={handleProviderToggle}
                onConfigure={handleProviderConfigure}
              />
            </Grid>
          ))}

        {/* Add Provider Card */}
        <Grid size={{ xs: 12, sm: 6, md: 4, lg: 3 }}>
          <ProviderCard
            isAddCard
            onConfigure={() => {
              // Open dialog without a provider to show the full list
              setProviderConfigDialog({ open: true, provider: null });
            }}
          />
        </Grid>
      </Grid>
    </Box>
  );

  const tabs = [
    {
      label: 'Providers',
      icon: <ProvidersIcon />,
      component: renderProvidersTab,
    },
    {
      label: 'General',
      icon: <GeneralIcon />,
      component: () => (
        <Suspense
          fallback={<LoadingComponent message="Loading General Settings..." />}
        >
          <GeneralSettings
            config={_config}
            onSave={saveSettings}
            backendAvailable={backendAvailable}
          />
        </Suspense>
      ),
    },
    {
      label: t('settings.language'),
      icon: <LanguageIcon />,
      component: () => (
        <Suspense fallback={<LoadingComponent message={t('common.loading')} />}>
          <Box sx={{ p: 3 }}>
            <Typography variant="h6" gutterBottom>
              {t('settings.title')}
            </Typography>
            <Grid container spacing={3}>
              <Grid item xs={12} md={6}>
                <Paper elevation={1} sx={{ p: 3 }}>
                  <Typography variant="h6" gutterBottom>
                    Interface Language
                  </Typography>
                  <Typography
                    variant="body2"
                    color="text.secondary"
                    sx={{ mb: 2 }}
                  >
                    Choose your preferred language for the user interface.
                  </Typography>
                  <LanguageSelector />
                </Paper>
              </Grid>
              <Grid item xs={12}>
                <Paper elevation={1} sx={{ p: 3 }}>
                  <Typography variant="h6" gutterBottom>
                    Language Profiles
                  </Typography>
                  <LanguageProfiles backendAvailable={backendAvailable} />
                </Paper>
              </Grid>
            </Grid>
          </Box>
        </Suspense>
      ),
    },
    {
      label: 'Database',
      icon: <DatabaseIcon />,
      component: () => (
        <Suspense
          fallback={<LoadingComponent message="Loading Database Settings..." />}
        >
          <DatabaseSettings
            config={_config}
            onSave={saveSettings}
            backendAvailable={backendAvailable}
          />
        </Suspense>
      ),
    },
    {
      label: 'Authentication',
      icon: <AuthIcon />,
      component: () => (
        <Suspense
          fallback={
            <LoadingComponent message="Loading Authentication Settings..." />
          }
        >
          <AuthSettings
            config={_config}
            onSave={saveSettings}
            backendAvailable={backendAvailable}
          />
        </Suspense>
      ),
    },
    {
      label: 'Notifications',
      icon: <NotificationIcon />,
      component: () => (
        <Suspense
          fallback={
            <LoadingComponent message="Loading Notification Settings..." />
          }
        >
          <NotificationSettings
            config={_config}
            onSave={saveSettings}
            backendAvailable={backendAvailable}
          />
        </Suspense>
      ),
    },
    {
      label: 'Users',
      icon: <UsersIcon />,
      component: () => (
        <Suspense
          fallback={<LoadingComponent message="Loading User Management..." />}
        >
          <UserManagement backendAvailable={backendAvailable} />
        </Suspense>
      ),
    },
    {
      label: 'Tags',
      icon: <TagsIcon />,
      component: () => (
        <Suspense
          fallback={<LoadingComponent message="Loading Tag Management..." />}
        >
          <TagManagement backendAvailable={backendAvailable} />
        </Suspense>
      ),
    },
    {
      label: 'Webhooks',
      icon: <WebhookIcon />,
      component: () => (
        <Suspense
          fallback={<LoadingComponent message="Loading Webhook Settings..." />}
        >
          <WebhookSettings backendAvailable={backendAvailable} />
        </Suspense>
      ),
    },
  ];

  const sectionIndex = tabs.findIndex(
    tab => tab.label.toLowerCase() === section
  );
  useEffect(() => {
    if (sectionIndex >= 0) {
      setActiveTab(sectionIndex);
    }
  }, [sectionIndex]);

  if (loading) {
    return (
      <Box
        display="flex"
        justifyContent="center"
        alignItems="center"
        minHeight="400px"
      >
        <CircularProgress />
      </Box>
    );
  }

  return (
    <Box>
      <Typography variant="h4" component="h1" gutterBottom>
        Settings
      </Typography>
      <BackButton />

      {/* Backend availability warning */}
      {!backendAvailable && (
        <Alert severity="error" sx={{ mb: 3 }}>
          Backend service is not available. Settings cannot be loaded or
          modified at this time.
        </Alert>
      )}

      {/* Error display */}
      {error && (
        <Alert severity="error" sx={{ mb: 3 }} onClose={() => setError(null)}>
          {error}
        </Alert>
      )}

      <Paper sx={{ mb: 3 }}>
        <Tabs
          value={activeTab}
          onChange={(e, newValue) => setActiveTab(newValue)}
          variant="scrollable"
          scrollButtons="auto"
        >
          {tabs.map((tab, index) => (
            <Tab
              key={index}
              label={tab.label}
              icon={tab.icon}
              iconPosition="start"
              disabled={!backendAvailable}
            />
          ))}
        </Tabs>
      </Paper>

      <Box>{tabs[activeTab]?.component()}</Box>

      {/* Provider Configuration Dialog */}
      <ProviderConfigDialog
        open={providerConfigDialog.open}
        provider={providerConfigDialog.provider}
        onClose={() => setProviderConfigDialog({ open: false, provider: null })}
        onSave={handleProviderSave}
      />

      {/* Bazarr Import Dialog */}
      <ImportDialog
        open={importDialogOpen}
        onClose={() => setImportDialogOpen(false)}
        bazarrURL={bazarrURL}
        setBazarrURL={setBazarrURL}
        bazarrAPIKey={bazarrAPIKey}
        setBazarrAPIKey={setBazarrAPIKey}
        bazarrMappings={bazarrMappings}
        selectedSettings={selectedSettings}
        setSelectedSettings={setSelectedSettings}
        previewConfig={previewConfig}
        importLoading={importLoading}
        importError={importError}
        importing={importing}
        onPreview={previewBazarr}
        onImport={importFromBazarr}
      />

      {/* Status Snackbar */}
      <Snackbar
        open={snackbarOpen}
        autoHideDuration={6000}
        onClose={() => setSnackbarOpen(false)}
      >
        <Alert
          onClose={() => setSnackbarOpen(false)}
          severity={
            status.includes('success') ||
            status.includes('enabled') ||
            status.includes('disabled') ||
            status.includes('saved')
              ? 'success'
              : 'error'
          }
          sx={{ width: '100%' }}
        >
          {status}
        </Alert>
      </Snackbar>
    </Box>
  );
}

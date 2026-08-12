// file: webui/src/components/ProviderConfigDialog.jsx
import {
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  Button,
  TextField,
  FormControl,
  InputLabel,
  Select,
  MenuItem,
  Box,
  Typography,
  Alert,
  FormControlLabel,
  Switch,
} from '@mui/material';
import { useEffect, useState } from 'react';
import { apiService } from '../services/api.js';

/**
 * Fixed ProviderConfigDialog with proper provider selection and configuration options
 *
 * @param {boolean} open - Whether dialog is open
 * @param {Object} provider - Provider object if editing existing provider
 * @param {Function} onClose - Callback when dialog closes
 * @param {Function} onSave - Callback when configuration is saved
 */
export default function ProviderConfigDialog({
  open,
  provider,
  onClose,
  onSave,
}) {
  const [selectedProvider, setSelectedProvider] = useState('');
  const [config, setConfig] = useState({});
  const [availableProviders, setAvailableProviders] = useState([]);

  // Load available providers when dialog opens
  useEffect(() => {
    if (!open) return;

    if (provider) {
      setSelectedProvider(provider.name);
      setConfig(provider.config || {});
    } else {
      setSelectedProvider('');
      setConfig({});
      loadAvailableProviders();
    }
  }, [open, provider]);

  const loadAvailableProviders = async () => {
    try {
      const response = await apiService.get('/api/providers/available');
      if (response?.ok) {
        const providers = await response.json();
        // Guard the shape rather than trusting it. /api/providers/available
        // has no handler registered on the server, so the request falls
        // through to the single-page-app catch-all and comes back as
        // index.html with a 200 — response.ok is true and the body is not an
        // array. Assigning it straight to state crashed the dialog on
        // `availableProviders.map is not a function`, which reads as a React
        // bug rather than a missing route.
        setAvailableProviders(Array.isArray(providers) ? providers : []);
      } else {
        setAvailableProviders([]);
      }
    } catch (error) {
      console.error('Failed to load available providers:', error);
      setAvailableProviders([]);
    }
  };

  const getProviderDisplayName = name => {
    const displayNames = {
      opensubtitles: 'OpenSubtitles.org',
      opensubtitlescom: 'OpenSubtitles.com',
      addic7ed: 'Addic7ed',
      subscene: 'Subscene',
      podnapisi: 'Podnapisi.NET',
      yifysubtitles: 'YIFY Subtitles',
      embedded: 'Embedded Subtitles',
      // Add more as needed
    };
    return displayNames[name] || name.charAt(0).toUpperCase() + name.slice(1);
  };

  const getProviderConfigFields = providerName => {
    const configs = {
      opensubtitles: [
        { key: 'api_key', label: 'API Key', type: 'password', required: true },
        {
          key: 'user_agent',
          label: 'User Agent',
          type: 'text',
          required: true,
        },
        { key: 'enabled', label: 'Enabled', type: 'boolean', default: true },
      ],
      opensubtitlescom: [
        { key: 'api_key', label: 'API Key', type: 'password', required: true },
        { key: 'enabled', label: 'Enabled', type: 'boolean', default: true },
      ],
      addic7ed: [
        { key: 'username', label: 'Username', type: 'text', required: true },
        {
          key: 'password',
          label: 'Password',
          type: 'password',
          required: true,
        },
        { key: 'enabled', label: 'Enabled', type: 'boolean', default: true },
      ],
      subscene: [
        { key: 'enabled', label: 'Enabled', type: 'boolean', default: true },
        {
          key: 'timeout',
          label: 'Timeout (seconds)',
          type: 'number',
          default: 30,
        },
      ],
      embedded: [
        { key: 'enabled', label: 'Enabled', type: 'boolean', default: true },
        {
          key: 'extract_mkv',
          label: 'Extract from MKV',
          type: 'boolean',
          default: true,
        },
        {
          key: 'extract_mp4',
          label: 'Extract from MP4',
          type: 'boolean',
          default: true,
        },
        {
          key: 'ffmpeg_path',
          label: 'FFmpeg Path',
          type: 'text',
          placeholder: '/usr/bin/ffmpeg',
        },
      ],
      // Add configurations for other providers
    };

    return (
      configs[providerName] || [
        { key: 'enabled', label: 'Enabled', type: 'boolean', default: true },
      ]
    );
  };

  const handleProviderChange = newProvider => {
    setSelectedProvider(newProvider);
    // Reset config when provider changes
    const fields = getProviderConfigFields(newProvider);
    const newConfig = {};
    fields.forEach(field => {
      if (field.default !== undefined) {
        newConfig[field.key] = field.default;
      }
    });
    setConfig(newConfig);
  };

  const handleConfigChange = (key, value) => {
    setConfig(prev => ({
      ...prev,
      [key]: value,
    }));
  };

  const handleSave = () => {
    if (!selectedProvider) {
      alert('Please select a provider');
      return;
    }

    const providerData = {
      name: selectedProvider,
      displayName: getProviderDisplayName(selectedProvider),
      config: config,
      enabled: config.enabled !== false,
    };

    // Only close once the save is known to have succeeded. This used to call
    // onSave and onClose back to back, so a failed save looked identical to a
    // successful one — the dialog shut and the operator moved on.
    //
    // A handler that returns nothing is treated as success, so callers that
    // do not report a result keep their old behaviour.
    Promise.resolve(onSave(providerData)).then(result => {
      if (result !== false) {
        onClose();
      }
    });
  };

  const renderConfigField = field => {
    const value = config[field.key] ?? field.default ?? '';

    switch (field.type) {
      case 'boolean':
        return (
          <FormControlLabel
            key={field.key}
            control={
              <Switch
                checked={!!value}
                onChange={e => handleConfigChange(field.key, e.target.checked)}
              />
            }
            label={field.label}
          />
        );

      case 'number':
        return (
          <TextField
            key={field.key}
            fullWidth
            label={field.label}
            type="number"
            value={value}
            onChange={e =>
              handleConfigChange(field.key, parseInt(e.target.value) || 0)
            }
            required={field.required}
            placeholder={field.placeholder}
            sx={{ mb: 2 }}
          />
        );

      case 'password':
        return (
          <TextField
            key={field.key}
            fullWidth
            label={field.label}
            type="password"
            value={value}
            onChange={e => handleConfigChange(field.key, e.target.value)}
            required={field.required}
            placeholder={field.placeholder}
            sx={{ mb: 2 }}
          />
        );

      default:
        return (
          <TextField
            key={field.key}
            fullWidth
            label={field.label}
            value={value}
            onChange={e => handleConfigChange(field.key, e.target.value)}
            required={field.required}
            placeholder={field.placeholder}
            sx={{ mb: 2 }}
          />
        );
    }
  };

  const isValid = () => {
    if (!selectedProvider) return false;

    const fields = getProviderConfigFields(selectedProvider);
    return fields.every(field => {
      if (field.required) {
        const value = config[field.key];
        return value !== undefined && value !== null && value !== '';
      }
      return true;
    });
  };

  return (
    <Dialog open={open} onClose={onClose} maxWidth="sm" fullWidth>
      <DialogTitle>
        {provider ? 'Configure Provider' : 'Configure Custom Provider'}
      </DialogTitle>

      <DialogContent>
        <Box sx={{ pt: 1 }}>
          {!provider && (
            <>
              <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>
                Configure this provider to enable subtitle downloads. Required
                fields are marked with an asterisk (*).
              </Typography>

              <FormControl fullWidth sx={{ mb: 3 }}>
                <InputLabel>Provider</InputLabel>
                <Select
                  value={selectedProvider}
                  label="Provider"
                  onChange={e => handleProviderChange(e.target.value)}
                >
                  {availableProviders.map(p => (
                    <MenuItem key={p.name} value={p.name}>
                      {getProviderDisplayName(p.name)}
                    </MenuItem>
                  ))}
                </Select>
              </FormControl>
            </>
          )}

          {(provider || selectedProvider) && (
            <>
              <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>
                Configure this provider to enable subtitle downloads. Required
                fields are marked with an asterisk (*).
              </Typography>

              <Alert severity="info" sx={{ mb: 2 }}>
                Provider:{' '}
                <strong>
                  {getProviderDisplayName(provider?.name || selectedProvider)}
                </strong>
              </Alert>

              <Box>
                {getProviderConfigFields(
                  provider?.name || selectedProvider
                ).map(renderConfigField)}
              </Box>
            </>
          )}
        </Box>
      </DialogContent>

      <DialogActions>
        <Button onClick={onClose}>Cancel</Button>
        <Button onClick={handleSave} variant="contained" disabled={!isValid()}>
          Save Configuration
        </Button>
      </DialogActions>
    </Dialog>
  );
}

// file: webui/src/ConfigEditor.jsx
import { Save as SaveIcon } from '@mui/icons-material';
import {
  Alert,
  Box,
  Button,
  CircularProgress,
  TextField,
  Typography,
} from '@mui/material';
import * as yaml from 'js-yaml';
import { useEffect, useState } from 'react';
import { apiService } from './services/api.js';

/**
 * ConfigEditor allows viewing and editing of the raw configuration
 * file using YAML format. Changes are persisted via the /api/config
 * endpoint which writes to Viper's configuration.
 *
 * @param {Object} props - Component props
 * @param {boolean} props.backendAvailable - Whether the backend service is available
 */
export default function ConfigEditor({ backendAvailable = true }) {
  const [yamlText, setYamlText] = useState('');
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [status, setStatus] = useState('');
  const [error, setError] = useState('');

  const saveConfig = async () => {
    if (!backendAvailable) return;
    setSaving(true);
    try {
      const obj = yaml.load(yamlText);
      const res = await apiService.post('/api/config', obj);
      if (res.ok) {
        setStatus('Configuration saved');
      } else {
        setError('Failed to save configuration');
      }
    } catch {
      setError('Invalid YAML or network error');
    } finally {
      setSaving(false);
    }
  };

  // useEffect for loading config
  useEffect(() => {
    const loadConfig = async () => {
      if (!backendAvailable) return;
      setLoading(true);
      try {
        const res = await apiService.get('/api/config');
        if (res.ok) {
          const data = await res.json();
          setYamlText(yaml.dump(data));
        } else {
          setError('Failed to load configuration');
        }
      } catch {
        setError('Failed to load configuration');
      } finally {
        setLoading(false);
      }
    };
    loadConfig();
  }, [backendAvailable]);

  return (
    <Box>
      <Typography variant="h4" component="h1" gutterBottom>
        Configuration File
      </Typography>

      {!backendAvailable && (
        <Alert severity="error" sx={{ mb: 3 }}>
          Backend service is not available. Configuration cannot be edited.
        </Alert>
      )}

      {loading ? (
        <Box
          display="flex"
          justifyContent="center"
          alignItems="center"
          minHeight="200px"
        >
          <CircularProgress />
        </Box>
      ) : (
        <Box>
          {error && (
            <Alert severity="error" sx={{ mb: 2 }} onClose={() => setError('')}>
              {error}
            </Alert>
          )}
          {status && (
            <Alert
              severity="success"
              sx={{ mb: 2 }}
              onClose={() => setStatus('')}
            >
              {status}
            </Alert>
          )}
          <TextField
            label="YAML Configuration"
            multiline
            minRows={20}
            value={yamlText}
            onChange={e => setYamlText(e.target.value)}
            fullWidth
            variant="outlined"
            disabled={!backendAvailable}
            sx={{
              fontFamily: '"Roboto Mono", "Consolas", "Monaco", monospace',
            }}
            data-testid="config-editor"
          />
          <Box mt={2}>
            <Button
              variant="contained"
              startIcon={<SaveIcon />}
              onClick={saveConfig}
              disabled={saving || !backendAvailable}
            >
              {saving ? 'Saving...' : 'Save'}
            </Button>
          </Box>
        </Box>
      )}
    </Box>
  );
}

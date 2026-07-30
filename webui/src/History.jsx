import {
  Download as DownloadIcon,
  FilterList as FilterIcon,
  Language as LanguageIcon,
  Translate as TranslateIcon,
} from '@mui/icons-material';
import {
  Alert,
  Avatar,
  Box,
  Card,
  CardContent,
  Chip,
  Paper,
  Tab,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  Tabs,
  TextField,
  Typography,
} from '@mui/material';
import { useEffect, useState } from 'react';
import { apiFetch } from './services/api.js';

/**
 * History displays translation and download history with optional language filtering.
 * Records are loaded from `/api/history` and filtered client side.
 * @param {Object} props - Component props
 * @param {boolean} props.backendAvailable - Whether the backend service is available
 */
export default function History({ backendAvailable = true }) {
  const [data, setData] = useState({ translations: [], downloads: [] });
  const [lang, setLang] = useState('');
  const [tabValue, setTabValue] = useState(0);

  useEffect(() => {
    if (backendAvailable) {
      apiFetch('/api/history')
        .then(r => {
          if (!r.ok) throw new Error(`HTTP ${r.status}`);
          return r.json();
        })
        .then(d =>
          setData({
            translations: d.translations || [],
            downloads: d.downloads || [],
          })
        )
        .catch(() => setData({ translations: [], downloads: [] }));
    }
  }, [backendAvailable]);

  const translations = (data.translations || []).filter(
    r => !lang || r.Language === lang
  );
  const downloads = (data.downloads || []).filter(
    r => !lang || r.Language === lang
  );

  const formatTimestamp = timestamp => {
    if (!timestamp) return 'N/A';
    return new Date(timestamp).toLocaleString();
  };

  return (
    <Box>
      {!backendAvailable && (
        <Alert severity="error" sx={{ mb: 3 }}>
          Backend service is not available. History data cannot be loaded.
        </Alert>
      )}

      <Box
        display="flex"
        justifyContent="space-between"
        alignItems="center"
        mb={3}
      >
        <Typography variant="h4" component="h1">
          History
        </Typography>
        <TextField
          size="small"
          placeholder="Filter by language (e.g., en, es, fr)"
          value={lang}
          onChange={e => setLang(e.target.value)}
          disabled={!backendAvailable}
          InputProps={{
            startAdornment: (
              <FilterIcon sx={{ mr: 1, color: 'action.active' }} />
            ),
          }}
          sx={{ minWidth: 250 }}
        />
      </Box>

      <Card>
        <Tabs
          value={tabValue}
          onChange={(e, newValue) => setTabValue(newValue)}
        >
          <Tab
            icon={<TranslateIcon />}
            label={`Translations (${translations.length})`}
            iconPosition="start"
          />
          <Tab
            icon={<DownloadIcon />}
            label={`Downloads (${downloads.length})`}
            iconPosition="start"
          />
        </Tabs>

        <CardContent>
          {tabValue === 0 ? (
            // Translations Tab
            <TableContainer component={Paper} variant="outlined">
              <Table>
                <TableHead>
                  <TableRow>
                    <TableCell>File</TableCell>
                    <TableCell>Language</TableCell>
                    <TableCell>Service</TableCell>
                    <TableCell>Status</TableCell>
                    <TableCell>Timestamp</TableCell>
                  </TableRow>
                </TableHead>
                <TableBody>
                  {translations.length === 0 ? (
                    <TableRow>
                      <TableCell colSpan={5} align="center">
                        <Typography color="text.secondary">
                          No translation history found
                        </Typography>
                      </TableCell>
                    </TableRow>
                  ) : (
                    translations.map(t => (
                      <TableRow key={t.ID} hover>
                        <TableCell>
                          <Typography
                            variant="body2"
                            sx={{ fontFamily: 'monospace' }}
                          >
                            {t.File}
                          </Typography>
                        </TableCell>
                        <TableCell>
                          <Box display="flex" alignItems="center">
                            <Avatar
                              sx={{
                                width: 24,
                                height: 24,
                                mr: 1,
                                fontSize: '0.75rem',
                              }}
                            >
                              <LanguageIcon fontSize="small" />
                            </Avatar>
                            <Chip label={t.Language} size="small" />
                          </Box>
                        </TableCell>
                        <TableCell>
                          <Chip
                            label={t.Service}
                            variant="outlined"
                            size="small"
                          />
                        </TableCell>
                        <TableCell>
                          <Chip
                            label={t.Status || 'Completed'}
                            color={t.Status === 'Error' ? 'error' : 'success'}
                            size="small"
                          />
                        </TableCell>
                        <TableCell>
                          <Typography variant="body2" color="text.secondary">
                            {formatTimestamp(t.Timestamp)}
                          </Typography>
                        </TableCell>
                      </TableRow>
                    ))
                  )}
                </TableBody>
              </Table>
            </TableContainer>
          ) : (
            // Downloads Tab
            <TableContainer component={Paper} variant="outlined">
              <Table>
                <TableHead>
                  <TableRow>
                    <TableCell>Video File</TableCell>
                    <TableCell>Language</TableCell>
                    <TableCell>Provider</TableCell>
                    <TableCell>Status</TableCell>
                    <TableCell>Timestamp</TableCell>
                  </TableRow>
                </TableHead>
                <TableBody>
                  {downloads.length === 0 ? (
                    <TableRow>
                      <TableCell colSpan={5} align="center">
                        <Typography color="text.secondary">
                          No download history found
                        </Typography>
                      </TableCell>
                    </TableRow>
                  ) : (
                    downloads.map(d => (
                      <TableRow key={d.ID} hover>
                        <TableCell>
                          <Typography
                            variant="body2"
                            sx={{ fontFamily: 'monospace' }}
                          >
                            {d.VideoFile}
                          </Typography>
                        </TableCell>
                        <TableCell>
                          <Box display="flex" alignItems="center">
                            <Avatar
                              sx={{
                                width: 24,
                                height: 24,
                                mr: 1,
                                fontSize: '0.75rem',
                              }}
                            >
                              <LanguageIcon fontSize="small" />
                            </Avatar>
                            <Chip label={d.Language} size="small" />
                          </Box>
                        </TableCell>
                        <TableCell>
                          <Chip
                            label={d.Provider}
                            variant="outlined"
                            size="small"
                          />
                        </TableCell>
                        <TableCell>
                          <Chip
                            label={d.Status || 'Downloaded'}
                            color={d.Status === 'Error' ? 'error' : 'success'}
                            size="small"
                          />
                        </TableCell>
                        <TableCell>
                          <Typography variant="body2" color="text.secondary">
                            {formatTimestamp(d.Timestamp)}
                          </Typography>
                        </TableCell>
                      </TableRow>
                    ))
                  )}
                </TableBody>
              </Table>
            </TableContainer>
          )}
        </CardContent>
      </Card>
    </Box>
  );
}

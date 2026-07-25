import {
  Add as AddIcon,
  Delete as DeleteIcon,
  Download as DownloadIcon,
  FilterList as FilterIcon,
  History as HistoryIcon,
  Language as LanguageIcon,
  Movie as MediaIcon,
  Preview as PreviewIcon,
  CloudDownload as ProviderIcon,
  Search as SearchIcon,
  Star as StarIcon,
  Visibility as ViewIcon,
  Download as WantedIcon,
} from '@mui/icons-material';
import {
  Alert,
  Autocomplete,
  Badge,
  Box,
  Button,
  Card,
  CardContent,
  Checkbox,
  Chip,
  CircularProgress,
  Collapse,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  FormControl,
  FormControlLabel,
  FormGroup,
  Grid,
  IconButton,
  InputLabel,
  List,
  ListItem,
  ListItemSecondaryAction,
  ListItemText,
  MenuItem,
  Paper,
  Rating,
  Select,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  TableSortLabel,
  TextField,
  Tooltip,
  Typography,
} from '@mui/material';
import { useEffect, useState } from 'react';

/**
 * Wanted provides an interface for searching subtitles and maintaining
 * a list of wanted items. Results are fetched from `/api/search` and
 * selections are POSTed to `/api/wanted`.
 * @param {Object} props - Component props
 * @param {boolean} props.backendAvailable - Whether the backend service is available
 */
export default function Wanted({ backendAvailable = true }) {
  // Enhanced state for comprehensive search functionality
  const [selectedProviders, setSelectedProviders] = useState([
    'opensubtitles',
    'subscene',
  ]);
  const [path, setPath] = useState('');
  const [lang, setLang] = useState('en');
  const [results, setResults] = useState([]);
  const [wanted, setWanted] = useState([]);
  const [wantedError, setWantedError] = useState('');
  const [searching, setSearching] = useState(false);
  const [loading, setLoading] = useState(true);

  // Advanced search filters
  const [season, setSeason] = useState('');
  const [episode, setEpisode] = useState('');
  const [year, setYear] = useState('');
  const [releaseGroup, setReleaseGroup] = useState('');
  const [showFilters, setShowFilters] = useState(false);

  // Table and UI state
  const [selectedResults, setSelectedResults] = useState([]);
  const [sortBy, setSortBy] = useState('score');
  const [sortOrder, setSortOrder] = useState('desc');
  const [previewDialog, setPreviewDialog] = useState({
    open: false,
    content: '',
    name: '',
    provider: '',
  });

  // Search history
  const [searchHistory, setSearchHistory] = useState([]);
  const [showHistory, setShowHistory] = useState(false);

  const providers = [
    { value: 'embedded', label: 'Embedded', available: true },
    { value: 'generic', label: 'Generic', available: true },
    { value: 'opensubtitles', label: 'OpenSubtitles', available: true },
    { value: 'addic7ed', label: 'Addic7ed', available: true },
    { value: 'podnapisi', label: 'Podnapisi', available: true },
    { value: 'subscene', label: 'Subscene', available: true },
    { value: 'bsplayer', label: 'BSPlayer', available: false },
    { value: 'tvsubtitles', label: 'TVSubtitles', available: false },
  ];

  const languages = [
    { code: 'en', name: 'English' },
    { code: 'es', name: 'Spanish' },
    { code: 'fr', name: 'French' },
    { code: 'de', name: 'German' },
    { code: 'it', name: 'Italian' },
    { code: 'pt', name: 'Portuguese' },
    { code: 'ru', name: 'Russian' },
    { code: 'ja', name: 'Japanese' },
    { code: 'ko', name: 'Korean' },
    { code: 'zh', name: 'Chinese' },
    { code: 'zh-cn', name: 'Chinese (Simplified)' },
    { code: 'zh-hans', name: 'Chinese (Simplified)' },
    { code: 'zh-tw', name: 'Chinese (Taiwan)' },
    { code: 'zh-hk', name: 'Chinese (Hong Kong)' },
    { code: 'zh-hant', name: 'Chinese (Traditional)' },
  ];

  useEffect(() => {
    const loadWanted = async () => {
      setLoading(true);
      try {
        const res = await fetch('/api/wanted');
        if (res.ok) {
          const data = await res.json();
          setWanted(data || []);
        }
      } catch (error) {
        console.error('Failed to load wanted list:', error);
      } finally {
        setLoading(false);
      }
    };
    loadWanted();
  }, []);

  const search = async () => {
    if (!path.trim()) {
      return;
    }

    if (selectedProviders.length === 0) {
      alert('Please select at least one provider');
      return;
    }

    setSearching(true);
    setResults([]);

    try {
      // Prepare search request
      const searchRequest = {
        providers: selectedProviders,
        mediaPath: path,
        language: lang,
        ...(season && { season: parseInt(season) }),
        ...(episode && { episode: parseInt(episode) }),
        ...(year && { year: parseInt(year) }),
        ...(releaseGroup && { releaseGroup }),
      };

      const response = await fetch('/api/search', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(searchRequest),
      });

      if (response.ok) {
        const data = await response.json();
        setResults(data.results || []);

        // Save to search history
        const historyItem = {
          query: searchRequest,
          timestamp: new Date(),
          results: data.results?.length || 0,
        };
        setSearchHistory(prev => [historyItem, ...prev.slice(0, 9)]); // Keep last 10 searches
      } else {
        console.error('Search failed:', response.statusText);
      }
    } catch (error) {
      console.error('Search failed:', error);
    } finally {
      setSearching(false);
    }
  };

  // The wanted list tracks *media files* that are missing subtitles, matching
  // Bazarr. It previously held subtitle download URLs bookmarked from search
  // results, which is a different concept that Bazarr has no equivalent of and
  // that no backend ever implemented.
  const reloadWanted = async () => {
    const res = await fetch('/api/wanted');
    if (res.ok) {
      setWanted((await res.json()) || []);
    }
  };

  const add = async (mediaPath, languages) => {
    try {
      const res = await fetch('/api/wanted', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ path: mediaPath, languages }),
      });
      if (res.ok) {
        await reloadWanted();
      } else {
        setWantedError(`Could not monitor ${mediaPath}`);
      }
    } catch (error) {
      console.error('Failed to add to wanted list:', error);
      setWantedError(`Could not monitor ${mediaPath}`);
    }
  };

  const remove = async mediaPath => {
    try {
      const res = await fetch('/api/wanted', {
        method: 'DELETE',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ path: mediaPath }),
      });
      if (res.ok) {
        setWanted(wanted.filter(item => item.path !== mediaPath));
      }
    } catch (error) {
      console.error('Failed to remove from wanted list:', error);
    }
  };

  const handleKeyPress = event => {
    if (event.key === 'Enter') {
      search();
    }
  };

  // Provider selection handlers
  const handleProviderToggle = providerValue => {
    setSelectedProviders(prev =>
      prev.includes(providerValue)
        ? prev.filter(p => p !== providerValue)
        : [...prev, providerValue]
    );
  };

  const handleSelectAllProviders = () => {
    const availableProviders = providers
      .filter(p => p.available)
      .map(p => p.value);
    setSelectedProviders(
      selectedProviders.length === availableProviders.length
        ? []
        : availableProviders
    );
  };

  // Result selection handlers
  const handleResultSelect = resultId => {
    setSelectedResults(prev =>
      prev.includes(resultId)
        ? prev.filter(id => id !== resultId)
        : [...prev, resultId]
    );
  };

  const handleSelectAllResults = () => {
    setSelectedResults(
      selectedResults.length === results.length ? [] : results.map(r => r.id)
    );
  };

  // Preview handler
  const handlePreview = async result => {
    try {
      const response = await fetch(
        `/api/search/preview?url=${encodeURIComponent(result.downloadUrl)}&provider=${result.provider}&lang=${result.language}`
      );
      if (response.ok) {
        const data = await response.json();
        setPreviewDialog({
          open: true,
          content: data.content,
          name: data.name || result.name,
          provider: data.provider || result.provider,
        });
      }
    } catch (error) {
      console.error('Preview failed:', error);
    }
  };

  // Batch download handler
  const handleBatchDownload = () => {
    const selectedItems = results.filter(r => selectedResults.includes(r.id));
    selectedItems.forEach(result => {
      add(result.downloadUrl);
    });
    setSelectedResults([]);
  };

  // Sort results
  const sortedResults = [...results].sort((a, b) => {
    let aValue = a[sortBy];
    let bValue = b[sortBy];

    if (typeof aValue === 'string') {
      aValue = aValue.toLowerCase();
      bValue = bValue.toLowerCase();
    }

    if (sortOrder === 'asc') {
      return aValue > bValue ? 1 : -1;
    } else {
      return aValue < bValue ? 1 : -1;
    }
  });

  // Search from history
  const searchFromHistory = historyItem => {
    const query = historyItem.query;
    setSelectedProviders(query.providers);
    setPath(query.mediaPath);
    setLang(query.language);
    setSeason(query.season?.toString() || '');
    setEpisode(query.episode?.toString() || '');
    setYear(query.year?.toString() || '');
    setReleaseGroup(query.releaseGroup || '');
    setShowHistory(false);
  };

  return (
    <Box>
      <Typography variant="h4" component="h1" gutterBottom>
        Subtitle Search & Wanted List
      </Typography>

      {!backendAvailable && (
        <Alert severity="error" sx={{ mb: 3 }}>
          Backend service is not available. Search and wanted list features are
          currently disabled.
        </Alert>
      )}

      <Typography variant="body1" color="text.secondary" paragraph>
        Search for subtitles across multiple providers and maintain a wanted
        list for automated downloads.
      </Typography>

      <Grid container spacing={3}>
        {/* Search Form */}
        <Grid size={{ xs: 12 }}>
          <Card>
            <CardContent>
              <Box
                display="flex"
                alignItems="center"
                justifyContent="space-between"
                mb={2}
              >
                <Typography variant="h6" display="flex" alignItems="center">
                  <SearchIcon sx={{ mr: 1 }} />
                  Manual Subtitle Search
                </Typography>
                <Box>
                  <Button
                    variant="outlined"
                    size="small"
                    startIcon={<HistoryIcon />}
                    onClick={() => setShowHistory(!showHistory)}
                    sx={{ mr: 1 }}
                  >
                    History
                  </Button>
                  <Button
                    variant="outlined"
                    size="small"
                    startIcon={<FilterIcon />}
                    onClick={() => setShowFilters(!showFilters)}
                  >
                    Filters
                  </Button>
                </Box>
              </Box>

              {/* Provider Selection */}
              <Box mb={3}>
                <Typography variant="subtitle2" gutterBottom>
                  Subtitle Providers
                </Typography>
                <Box display="flex" alignItems="center" mb={1}>
                  <FormControlLabel
                    control={
                      <Checkbox
                        checked={
                          selectedProviders.length ===
                          providers.filter(p => p.available).length
                        }
                        indeterminate={
                          selectedProviders.length > 0 &&
                          selectedProviders.length <
                            providers.filter(p => p.available).length
                        }
                        onChange={handleSelectAllProviders}
                      />
                    }
                    label="Select All"
                  />
                </Box>
                <FormGroup row>
                  {providers.map(provider => (
                    <FormControlLabel
                      key={provider.value}
                      control={
                        <Checkbox
                          checked={selectedProviders.includes(provider.value)}
                          onChange={() => handleProviderToggle(provider.value)}
                          disabled={!provider.available}
                        />
                      }
                      label={
                        <Box display="flex" alignItems="center">
                          {provider.label}
                          {!provider.available && (
                            <Chip
                              label="Config Required"
                              size="small"
                              color="warning"
                              sx={{ ml: 1 }}
                            />
                          )}
                        </Box>
                      }
                    />
                  ))}
                </FormGroup>
              </Box>

              {/* Main Search Fields */}
              <Grid container spacing={2} alignItems="center">
                <Grid size={{ xs: 12, sm: 6 }}>
                  <TextField
                    fullWidth
                    label="Media File Path"
                    placeholder="/path/to/movie.mkv"
                    value={path}
                    onChange={e => setPath(e.target.value)}
                    onKeyPress={handleKeyPress}
                    disabled={searching}
                    InputProps={{
                      startAdornment: (
                        <MediaIcon sx={{ mr: 1, color: 'action.active' }} />
                      ),
                    }}
                  />
                </Grid>
                <Grid size={{ xs: 12, sm: 3 }}>
                  <FormControl fullWidth>
                    <InputLabel>Language</InputLabel>
                    <Select
                      value={lang}
                      label="Language"
                      onChange={e => setLang(e.target.value)}
                      disabled={searching}
                      startAdornment={
                        <LanguageIcon sx={{ mr: 1, color: 'action.active' }} />
                      }
                    >
                      {languages.map(l => (
                        <MenuItem key={l.code} value={l.code}>
                          <Box display="flex" alignItems="center">
                            <Chip
                              label={l.code.toUpperCase()}
                              size="small"
                              sx={{ mr: 1, minWidth: 40 }}
                            />
                            {l.name}
                          </Box>
                        </MenuItem>
                      ))}
                    </Select>
                  </FormControl>
                </Grid>
                <Grid size={{ xs: 12, sm: 3 }}>
                  <Button
                    variant="contained"
                    startIcon={
                      searching ? (
                        <CircularProgress size={20} />
                      ) : (
                        <SearchIcon />
                      )
                    }
                    onClick={search}
                    disabled={
                      searching ||
                      !path.trim() ||
                      selectedProviders.length === 0
                    }
                    fullWidth
                    size="large"
                  >
                    {searching ? 'Searching...' : 'Search'}
                  </Button>
                  <Button
                    variant="outlined"
                    startIcon={<AddIcon />}
                    onClick={() => add(path.trim(), [lang])}
                    disabled={
                      !path.trim() ||
                      wanted.some(item => item.path === path.trim())
                    }
                    fullWidth
                    size="small"
                    sx={{ mt: 1 }}
                  >
                    {wanted.some(item => item.path === path.trim())
                      ? 'Monitored'
                      : 'Monitor this file'}
                  </Button>
                </Grid>
              </Grid>

              {/* Advanced Filters */}
              <Collapse in={showFilters}>
                <Box mt={3} pt={2} borderTop={1} borderColor="divider">
                  <Typography variant="subtitle2" gutterBottom>
                    Advanced Filters
                  </Typography>
                  <Grid container spacing={2}>
                    <Grid size={{ xs: 6, sm: 3 }}>
                      <TextField
                        fullWidth
                        label="Season"
                        type="number"
                        value={season}
                        onChange={e => setSeason(e.target.value)}
                        size="small"
                      />
                    </Grid>
                    <Grid size={{ xs: 6, sm: 3 }}>
                      <TextField
                        fullWidth
                        label="Episode"
                        type="number"
                        value={episode}
                        onChange={e => setEpisode(e.target.value)}
                        size="small"
                      />
                    </Grid>
                    <Grid size={{ xs: 6, sm: 3 }}>
                      <TextField
                        fullWidth
                        label="Year"
                        type="number"
                        value={year}
                        onChange={e => setYear(e.target.value)}
                        size="small"
                      />
                    </Grid>
                    <Grid size={{ xs: 6, sm: 3 }}>
                      <TextField
                        fullWidth
                        label="Release Group"
                        value={releaseGroup}
                        onChange={e => setReleaseGroup(e.target.value)}
                        size="small"
                      />
                    </Grid>
                  </Grid>
                </Box>
              </Collapse>

              {/* Search History */}
              <Collapse in={showHistory}>
                <Box mt={3} pt={2} borderTop={1} borderColor="divider">
                  <Typography variant="subtitle2" gutterBottom>
                    Recent Searches
                  </Typography>
                  {searchHistory.length === 0 ? (
                    <Alert severity="info">No search history available</Alert>
                  ) : (
                    <List dense>
                      {searchHistory.slice(0, 5).map((item, index) => (
                        <ListItem
                          key={index}
                          button
                          onClick={() => searchFromHistory(item)}
                          divider={index < searchHistory.length - 1}
                        >
                          <ListItemText
                            primary={item.query.mediaPath}
                            secondary={`${item.results} results • ${item.timestamp.toLocaleString()}`}
                          />
                          <Chip
                            label={item.query.providers.join(', ')}
                            size="small"
                            variant="outlined"
                          />
                        </ListItem>
                      ))}
                    </List>
                  )}
                </Box>
              </Collapse>
            </CardContent>
          </Card>
        </Grid>

        {/* Search Results */}
        <Grid size={{ xs: 12, md: 8 }}>
          <Card>
            <CardContent>
              <Box
                display="flex"
                alignItems="center"
                justifyContent="space-between"
                mb={2}
              >
                <Typography variant="h6">
                  Search Results ({results.length})
                </Typography>
                {results.length > 0 && (
                  <Box>
                    <FormControlLabel
                      control={
                        <Checkbox
                          checked={selectedResults.length === results.length}
                          indeterminate={
                            selectedResults.length > 0 &&
                            selectedResults.length < results.length
                          }
                          onChange={handleSelectAllResults}
                        />
                      }
                      label="Select All"
                    />
                    {selectedResults.length > 0 && (
                      <Button
                        variant="contained"
                        size="small"
                        startIcon={<DownloadIcon />}
                        onClick={handleBatchDownload}
                        sx={{ ml: 1 }}
                      >
                        Download Selected ({selectedResults.length})
                      </Button>
                    )}
                  </Box>
                )}
              </Box>

              {searching ? (
                <Box display="flex" justifyContent="center" p={3}>
                  <CircularProgress />
                </Box>
              ) : results.length === 0 ? (
                <Alert severity="info">
                  No search results. Try searching for a media file to find
                  subtitles.
                </Alert>
              ) : (
                <TableContainer component={Paper} variant="outlined">
                  <Table size="small">
                    <TableHead>
                      <TableRow>
                        <TableCell padding="checkbox">
                          <Checkbox
                            checked={selectedResults.length === results.length}
                            indeterminate={
                              selectedResults.length > 0 &&
                              selectedResults.length < results.length
                            }
                            onChange={handleSelectAllResults}
                          />
                        </TableCell>
                        <TableCell>
                          <TableSortLabel
                            active={sortBy === 'score'}
                            direction={sortBy === 'score' ? sortOrder : 'asc'}
                            onClick={() => {
                              if (sortBy === 'score') {
                                setSortOrder(
                                  sortOrder === 'asc' ? 'desc' : 'asc'
                                );
                              } else {
                                setSortBy('score');
                                setSortOrder('desc');
                              }
                            }}
                          >
                            Score
                          </TableSortLabel>
                        </TableCell>
                        <TableCell>Provider</TableCell>
                        <TableCell>Name</TableCell>
                        <TableCell>Language</TableCell>
                        <TableCell>
                          <TableSortLabel
                            active={sortBy === 'downloads'}
                            direction={
                              sortBy === 'downloads' ? sortOrder : 'asc'
                            }
                            onClick={() => {
                              if (sortBy === 'downloads') {
                                setSortOrder(
                                  sortOrder === 'asc' ? 'desc' : 'asc'
                                );
                              } else {
                                setSortBy('downloads');
                                setSortOrder('desc');
                              }
                            }}
                          >
                            Downloads
                          </TableSortLabel>
                        </TableCell>
                        <TableCell>Actions</TableCell>
                      </TableRow>
                    </TableHead>
                    <TableBody>
                      {sortedResults.map(result => (
                        <TableRow
                          key={result.id}
                          hover
                          selected={selectedResults.includes(result.id)}
                        >
                          <TableCell padding="checkbox">
                            <Checkbox
                              checked={selectedResults.includes(result.id)}
                              onChange={() => handleResultSelect(result.id)}
                            />
                          </TableCell>
                          <TableCell>
                            <Box display="flex" alignItems="center">
                              <Rating
                                value={result.score}
                                max={1}
                                precision={0.1}
                                size="small"
                                readOnly
                              />
                              <Typography variant="caption" sx={{ ml: 1 }}>
                                {(result.score * 100).toFixed(0)}%
                              </Typography>
                            </Box>
                          </TableCell>
                          <TableCell>
                            <Chip
                              label={result.provider}
                              size="small"
                              variant="outlined"
                              color={result.fromTrusted ? 'success' : 'default'}
                            />
                          </TableCell>
                          <TableCell>
                            <Typography variant="body2" noWrap>
                              {result.name}
                              {result.isHI && (
                                <Chip label="HI" size="small" sx={{ ml: 1 }} />
                              )}
                            </Typography>
                          </TableCell>
                          <TableCell>
                            <Chip
                              label={result.language.toUpperCase()}
                              size="small"
                              sx={{ minWidth: 40 }}
                            />
                          </TableCell>
                          <TableCell>
                            <Typography variant="body2">
                              {result.downloads
                                ? result.downloads.toLocaleString()
                                : '-'}
                            </Typography>
                          </TableCell>
                          <TableCell>
                            <Box display="flex" gap={1}>
                              <Tooltip title="Preview">
                                <IconButton
                                  size="small"
                                  onClick={() => handlePreview(result)}
                                >
                                  <PreviewIcon fontSize="small" />
                                </IconButton>
                              </Tooltip>
                            </Box>
                          </TableCell>
                        </TableRow>
                      ))}
                    </TableBody>
                  </Table>
                </TableContainer>
              )}
            </CardContent>
          </Card>
        </Grid>

        {/* Wanted List */}
        <Grid size={{ xs: 12, md: 4 }}>
          <Card>
            <CardContent>
              <Typography variant="h6" gutterBottom>
                <WantedIcon sx={{ mr: 1, verticalAlign: 'middle' }} />
                Wanted List ({wanted.length})
              </Typography>
              {wantedError && (
                <Alert
                  severity="error"
                  onClose={() => setWantedError('')}
                  sx={{ mb: 1 }}
                >
                  {wantedError}
                </Alert>
              )}
              {loading ? (
                <Box display="flex" justifyContent="center" p={2}>
                  <CircularProgress size={24} />
                </Box>
              ) : wanted.length === 0 ? (
                <Alert severity="info">
                  Nothing is being monitored. Use &ldquo;Monitor this
                  file&rdquo; to have subtitles searched for automatically.
                </Alert>
              ) : (
                <Paper
                  variant="outlined"
                  sx={{ maxHeight: 400, overflow: 'auto' }}
                >
                  <List dense>
                    {wanted.map((item, index) => (
                      <ListItem
                        key={item.id || item.path}
                        divider={index < wanted.length - 1}
                      >
                        <ListItemText
                          primary={
                            <Tooltip title={item.path}>
                              <Typography variant="body2" noWrap>
                                {item.path.split('/').pop()}
                              </Typography>
                            </Tooltip>
                          }
                          secondary={
                            <Box
                              component="span"
                              display="flex"
                              gap={0.5}
                              alignItems="center"
                              flexWrap="wrap"
                              mt={0.5}
                            >
                              {(item.languages || []).map(code => (
                                <Chip
                                  key={code}
                                  label={code.toUpperCase()}
                                  size="small"
                                />
                              ))}
                              <Chip
                                label={item.status}
                                size="small"
                                color={
                                  item.status === 'found'
                                    ? 'success'
                                    : item.status === 'failed' ||
                                        item.status === 'blacklisted'
                                      ? 'error'
                                      : 'default'
                                }
                              />
                              {item.retry_count > 0 && (
                                <Typography
                                  variant="caption"
                                  color="text.secondary"
                                >
                                  {item.retry_count}/{item.max_retries} tries
                                </Typography>
                              )}
                            </Box>
                          }
                        />
                        <ListItemSecondaryAction>
                          <IconButton
                            edge="end"
                            size="small"
                            color="error"
                            onClick={() => remove(item.path)}
                          >
                            <DeleteIcon fontSize="small" />
                          </IconButton>
                        </ListItemSecondaryAction>
                      </ListItem>
                    ))}
                  </List>
                </Paper>
              )}
            </CardContent>
          </Card>
        </Grid>
      </Grid>

      {/* Preview Dialog */}
      <Dialog
        open={previewDialog.open}
        onClose={() => setPreviewDialog(prev => ({ ...prev, open: false }))}
        maxWidth="md"
        fullWidth
      >
        <DialogTitle>Subtitle Preview - {previewDialog.name}</DialogTitle>
        <DialogContent>
          <Box mb={2}>
            <Chip
              label={previewDialog.provider}
              size="small"
              variant="outlined"
            />
          </Box>
          <Paper
            variant="outlined"
            sx={{ p: 2, maxHeight: 400, overflow: 'auto' }}
          >
            <Typography
              variant="body2"
              component="pre"
              sx={{
                fontFamily: 'monospace',
                whiteSpace: 'pre-wrap',
                wordBreak: 'break-word',
              }}
            >
              {previewDialog.content}
            </Typography>
          </Paper>
        </DialogContent>
        <DialogActions>
          <Button
            onClick={() => setPreviewDialog(prev => ({ ...prev, open: false }))}
          >
            Close
          </Button>
        </DialogActions>
      </Dialog>
    </Box>
  );
}

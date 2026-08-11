// file: webui/src/MediaLibrary.jsx
// version: 2.3.0
// guid: 1a2b3c4d-5e6f-7a8b-9c0d-1e2f3a4b5c6d
// last-edited: 2026-08-11
// @ts-nocheck

import {
  Add as AddIcon,
  Folder as FolderIcon,
  GridView as GridViewIcon,
  List as ListIcon,
  MoreVert as MoreIcon,
  Movie as MovieIcon,
  QrCodeScanner as Scanner,
  Storage as StorageIcon,
  Subtitles as SubtitleIcon,
  Sync as SyncIcon,
} from '@mui/icons-material';
import {
  Alert,
  Box,
  Breadcrumbs,
  Button,
  Card,
  CardActionArea,
  CardContent,
  CardMedia,
  Checkbox,
  CircularProgress,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  FormControl,
  FormControlLabel,
  Grid,
  IconButton,
  InputLabel,
  Link,
  MenuItem,
  Paper,
  Select,
  Tab,
  Tabs,
  TextField,
  ToggleButton,
  ToggleButtonGroup,
  Typography,
} from '@mui/material';
import { useEffect, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { apiFetch } from './services/api.js';

/**
 * Video containers the library lists and mass edit may act on.
 */
const MEDIA_EXTENSIONS = /\.(mp4|mkv|avi|mov|wmv|flv|webm|m4v)$/i;

/**
 * True for a playable media file — not a directory, and a known video container.
 *
 * Shared by the listing filter and by "Select all files" so the two cannot
 * drift. They previously used different tests: the listing hid sidecar
 * subtitles while the selection swept them in, so two visible episodes
 * reported "4 assigned" and attached a language profile to two .srt files.
 *
 * @param {Object} item - A media item from GET /api/library/browse.
 * @returns {boolean} Whether the item is a media file.
 */
const isMediaFile = item =>
  Boolean(item) && !item.isDirectory && MEDIA_EXTENSIONS.test(item.name || '');

/**
 * MediaLibrary provides integrated media file and subtitle management.
 * Shows media files with their available subtitles, allows searching,
 * downloading, extracting, and translating subtitles directly from the file view.
 * @param {Object} props - Component props
 * @param {boolean} props.backendAvailable - Whether the backend service is available
 */
export default function MediaLibrary({ backendAvailable = true }) {
  const [currentPath, setCurrentPath] = useState('/');
  const [items, setItems] = useState([]);
  const [loading, setLoading] = useState(false);
  const [actionMenu, setActionMenu] = useState({ anchor: null, file: null });
  const [bulkMode, setBulkMode] = useState(false);
  const [error, setError] = useState(null);
  const [selectedFiles, setSelectedFiles] = useState(new Set());
  const [operationDialog, setOperationDialog] = useState({
    open: false,
    type: null,
    file: null,
  });
  const [progress, setProgress] = useState(null);
  const [tasks, setTasks] = useState({});
  // View mode: list, poster, or grid
  const [viewMode, setViewMode] = useState('list');
  // Active tab for Sonarr-style navigation
  const [activeTab, setActiveTab] = useState(0);
  // Library management
  const [addLibraryDialog, setAddLibraryDialog] = useState(false);
  const [newLibraryPath, setNewLibraryPath] = useState('');
  const [libraryPaths, setLibraryPaths] = useState([]);
  // Mass edit: the language profiles available to assign, the one chosen in the
  // toolbar, and the outcome of the last bulk request. bulkResult is kept so a
  // partial success stays on screen — the endpoint reports per-item outcomes
  // and collapsing that to a toast would hide which files did not take.
  const [languageProfiles, setLanguageProfiles] = useState([]);
  const [bulkProfileId, setBulkProfileId] = useState('');
  const [bulkResult, setBulkResult] = useState(null);
  // Combine (bilingual "double subs"): which subtitles are ticked per media
  // file, keyed by the file's path. Selection ORDER is meaningful — the first
  // pick becomes the primary language and renders on top of the stacked cue.
  const [subtitleSelection, setSubtitleSelection] = useState({});
  const [combineResult, setCombineResult] = useState(null);
  const [combineError, setCombineError] = useState(null);
  const navigate = useNavigate();

  // Fetch poster and basic details from OMDb
  const usePoster = title => {
    const [info, setInfo] = useState(null);
    useEffect(() => {
      let ignore = false;
      const load = async () => {
        try {
          const res = await fetch(
            `https://www.omdbapi.com/?t=${encodeURIComponent(title)}&apikey=thewdb`
          );
          if (res.ok) {
            const data = await res.json();
            if (!ignore && data.Response === 'True') {
              setInfo(data);
            }
          }
        } catch {
          // Ignore errors
        }
      };
      load();
      return () => {
        ignore = true;
      };
    }, [title]);
    return info;
  };

  // Component for grid view items that can use hooks
  const GridItem = ({ item }) => {
    const info = usePoster(item?.name || '');
    const poster =
      info?.Poster && info.Poster !== 'N/A'
        ? info.Poster
        : 'https://via.placeholder.com/300x450?text=Poster';

    return (
      <Grid item xs={6} md={3} key={`grid-item-${item?.path || Math.random()}`}>
        <CardActionArea
          onClick={() =>
            navigate(
              `/details?title=${encodeURIComponent(item?.name || '')}&path=${encodeURIComponent(item?.path || '')}`
            )
          }
        >
          <Card sx={{ height: '100%' }}>
            <CardMedia
              component="img"
              image={poster}
              sx={{ height: 450, objectFit: 'cover' }}
            />
            <CardContent sx={{ p: 1 }}>
              <Typography variant="body2" noWrap>
                {item?.name || 'Unknown'}
              </Typography>
              {info && (
                <Typography variant="caption" color="text.secondary">
                  {info.Year} • {info.Genre}
                </Typography>
              )}
            </CardContent>
          </Card>
        </CardActionArea>
      </Grid>
    );
  };

  // Component for poster view items that can use hooks
  const PosterItem = ({ item }) => {
    const info = usePoster(item?.name || '');
    const poster =
      info?.Poster && info.Poster !== 'N/A'
        ? info.Poster
        : 'https://via.placeholder.com/150x225?text=Poster';

    return (
      <Grid item xs={12} md={6}>
        <CardActionArea
          onClick={() =>
            navigate(
              `/details?title=${encodeURIComponent(item?.name || '')}&path=${encodeURIComponent(item?.path || '')}`
            )
          }
        >
          <Card sx={{ display: 'flex' }}>
            <CardMedia component="img" image={poster} sx={{ width: 150 }} />
            <CardContent>
              <Typography variant="h6" gutterBottom>
                {info?.Title || item?.name || 'Unknown'}
              </Typography>
              {info?.Plot && (
                <Typography variant="body2" color="text.secondary">
                  {info.Plot}
                </Typography>
              )}
              {info?.imdbRating && (
                <Typography
                  variant="body2"
                  color="text.secondary"
                  sx={{ mt: 1 }}
                >
                  IMDB: {info.imdbRating}
                </Typography>
              )}
            </CardContent>
          </Card>
        </CardActionArea>
      </Grid>
    );
  };

  const loadCurrentDirectory = async () => {
    setLoading(true);
    try {
      const response = await apiFetch(
        `/api/library/browse?path=${encodeURIComponent(currentPath)}`
      );
      if (response.ok) {
        const data = await response.json();
        // Normalise the directory flag once, here at the boundary, rather than
        // at each of the nine places that read it. The server marshals
        // `isDirectory` (see MediaItem in pkg/webserver/server.go); this
        // component previously read `is_dir`, which is always undefined, so
        // every directory was filtered out and the page rendered blank.
        setItems(
          (data.items || []).map(item => ({
            ...item,
            isDirectory: Boolean(item?.isDirectory),
          }))
        );
      } else {
        setItems([]);
      }
    } catch (error) {
      console.error('Failed to load directory:', error);
    } finally {
      setLoading(false);
    }
  };

  /**
   * Load the language profiles offered by the mass-edit toolbar.
   *
   * Uses /api/language-profiles, the spelling the rest of the frontend uses;
   * the server aliases it onto /api/profiles.
   */
  const loadLanguageProfiles = async () => {
    try {
      const response = await apiFetch('/api/language-profiles');
      if (response.ok) {
        const data = await response.json();
        setLanguageProfiles(Array.isArray(data) ? data : []);
      }
    } catch (error) {
      console.error('Failed to load language profiles:', error);
    }
  };

  useEffect(() => {
    loadCurrentDirectory();
    loadLibraryPaths();
    loadLanguageProfiles();

    // Set up task polling if backend is available
    if (backendAvailable) {
      const pollTasks = async () => {
        try {
          const response = await apiFetch('/api/tasks');
          if (response.ok) {
            const data = await response.json();
            setTasks(data || {});
          }
        } catch (error) {
          console.error('Failed to poll tasks:', error);
        }
      };

      pollTasks();
      const taskInterval = setInterval(pollTasks, 2000);
      return () => clearInterval(taskInterval);
    }
  }, [currentPath, backendAvailable]); // eslint-disable-line react-hooks/exhaustive-deps

  /**
   * Load configured library paths
   */
  const loadLibraryPaths = async () => {
    if (!backendAvailable) return;

    try {
      const response = await apiFetch('/api/library/paths');
      if (response.ok) {
        const data = await response.json();
        setLibraryPaths(data.paths || []);
      }
    } catch (error) {
      console.error('Failed to load library paths:', error);
    }
  };

  /**
   * Add new library path
   */
  const handleTabChange = (event, newValue) => {
    setActiveTab(newValue);
  };

  const handleAddLibraryPath = async () => {
    if (!newLibraryPath.trim()) return;

    try {
      const response = await apiFetch('/api/library/paths', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ path: newLibraryPath.trim() }),
      });

      if (response.ok) {
        await loadLibraryPaths();
        setNewLibraryPath('');
        setAddLibraryDialog(false);
        // Refresh current view if we're in root
        if (currentPath === '/') {
          await loadCurrentDirectory();
        }
      } else {
        setError('Failed to add library path');
      }
    } catch (error) {
      console.error('Failed to add library path:', error);
      setError('Failed to add library path');
    }
  };

  /**
   * Resync from Sonarr/Radarr
   */
  const handleResyncFromSonarr = async () => {
    try {
      setLoading(true);
      const response = await apiFetch('/api/library/resync', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ path: currentPath }),
      });

      if (response.ok) {
        await loadCurrentDirectory();
      } else {
        setError('Failed to resync from Sonarr/Radarr');
      }
    } catch (error) {
      console.error('Failed to resync from Sonarr/Radarr:', error);
      setError('Failed to resync from Sonarr/Radarr');
    } finally {
      setLoading(false);
    }
  };

  // Tab content based on active tab
  const getTabContent = () => {
    switch (activeTab) {
      case 0: // All Media
        return items;
      case 1: // Movies Only
        return items.filter(
          item =>
            item.type === 'movie' ||
            (!item.type && item.name?.match(/\.(mp4|mkv|avi|mov)$/i))
        );
      case 2: // TV Shows Only
        return items.filter(
          item => item.type === 'tv' || (!item.type && item.isDirectory)
        );
      case 3: // Library Paths
        return null; // Special case for library management
      default:
        return items;
    }
  };

  const renderLibraryPathsTab = () => (
    <Box sx={{ p: 3 }}>
      <Typography variant="h5" gutterBottom>
        Library Management
      </Typography>
      <Grid container spacing={2}>
        {libraryPaths.map((path, index) => (
          <Grid item xs={12} md={6} key={index}>
            <Card>
              <CardContent>
                <Typography variant="h6">{path}</Typography>
                <Box sx={{ mt: 2 }}>
                  <Button
                    startIcon={<Scanner />}
                    variant="outlined"
                    color="primary"
                    size="small"
                    sx={{ mr: 1 }}
                  >
                    Rescan Path
                  </Button>
                  <Button color="error" size="small">
                    Remove
                  </Button>
                </Box>
              </CardContent>
            </Card>
          </Grid>
        ))}
        <Grid item xs={12}>
          <Button
            startIcon={<AddIcon />}
            variant="contained"
            color="primary"
            onClick={() => setAddLibraryDialog(true)}
          >
            Add Library Path
          </Button>
        </Grid>
      </Grid>
    </Box>
  );

  const renderListView = itemsToRender => (
    <Grid container spacing={2}>
      {itemsToRender
        .filter(
          item =>
            item.isDirectory ||
            item.name?.match(/\.(mp4|mkv|avi|mov|wmv|flv|webm|m4v)$/i)
        )
        .map(item => (
          <Grid item xs={12} key={item.path || Math.random()}>
            <Card
              sx={{
                cursor: item.isDirectory ? 'pointer' : 'default',
                '&:hover': {
                  backgroundColor: 'action.hover',
                },
              }}
              onClick={() => {
                if (item.isDirectory) {
                  // Set the path and let the effect reload. Calling
                  // loadCurrentDirectory() here as well refetched the
                  // *previous* directory, because the call closed over the
                  // currentPath from this render.
                  navigateToPath(item.path);
                } else {
                  navigate(
                    `/details?title=${encodeURIComponent(item.name || '')}&path=${encodeURIComponent(item.path || '')}`
                  );
                }
              }}
            >
              <CardContent>
                <Box display="flex" alignItems="center">
                  {/*
                    Selection is offered only in mass-edit mode and only for
                    files: directories have no profile assignment, and showing a
                    checkbox that silently does nothing is worse than none.
                    The click must not bubble — the enclosing Card navigates.
                  */}
                  {bulkMode && !item.isDirectory && (
                    <Checkbox
                      checked={selectedFiles.has(item.path)}
                      onClick={event => event.stopPropagation()}
                      onChange={() => toggleFileSelection(item.path)}
                      inputProps={{
                        'aria-label': `Select ${item.name || 'file'}`,
                      }}
                    />
                  )}
                  <Box sx={{ mr: 2 }}>
                    {item.isDirectory ? (
                      <FolderIcon color="primary" />
                    ) : (
                      <MovieIcon color="action" />
                    )}
                  </Box>
                  <Box flex={1}>
                    <Typography variant="h6" noWrap>
                      {item.name || 'Unknown'}
                    </Typography>
                    {item.size && (
                      <Typography variant="body2" color="text.secondary">
                        {(item.size / 1024 / 1024 / 1024).toFixed(1)} GB
                      </Typography>
                    )}
                  </Box>
                  {!item.isDirectory && (
                    <IconButton size="small">
                      <MoreIcon />
                    </IconButton>
                  )}
                </Box>

                {/*
                  The browse endpoint has always reported each media file's
                  sidecars; the UI simply never showed them. Ticking two and
                  pressing Combine produces a bilingual "double subs" file.
                  Clicks must not bubble — the enclosing Card navigates.
                */}
                {!item.isDirectory && item.subtitles?.length > 0 && (
                  <Box sx={{ pl: 5, pt: 1 }} onClick={e => e.stopPropagation()}>
                    <Box
                      display="flex"
                      alignItems="center"
                      flexWrap="wrap"
                      gap={1}
                    >
                      {item.subtitles.map(subtitle => (
                        <FormControlLabel
                          key={subtitle.path}
                          control={
                            <Checkbox
                              size="small"
                              checked={(
                                subtitleSelection[item.path] || []
                              ).includes(subtitle.path)}
                              onChange={() =>
                                toggleSubtitleSelection(
                                  item.path,
                                  subtitle.path
                                )
                              }
                              inputProps={{
                                'aria-label': `Select subtitle ${subtitle.language}`,
                              }}
                            />
                          }
                          label={
                            <Typography variant="body2">
                              {subtitle.language}
                            </Typography>
                          }
                        />
                      ))}
                      <Button
                        size="small"
                        variant="outlined"
                        aria-label={`Combine subtitles for ${item.name}`}
                        disabled={
                          (subtitleSelection[item.path] || []).length !== 2
                        }
                        onClick={() => combineSubtitles(item)}
                      >
                        Combine
                      </Button>
                    </Box>
                    {(subtitleSelection[item.path] || []).length === 2 && (
                      <Typography variant="caption" color="text.secondary">
                        First pick renders on top of each cue.
                      </Typography>
                    )}
                  </Box>
                )}
              </CardContent>
            </Card>
          </Grid>
        ))}
    </Grid>
  );

  /**
   * Rescan library disk
   */
  const handleRescanDisk = async () => {
    try {
      setProgress({ type: 'rescan', file: 'library', progress: 0 });
      const response = await apiFetch('/api/library/rescan', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ path: currentPath }),
      });

      if (response.ok) {
        await loadCurrentDirectory();
      } else {
        setError('Failed to rescan disk');
      }
    } catch (error) {
      console.error('Failed to rescan disk:', error);
      setError('Failed to rescan disk');
    } finally {
      setProgress(null);
    }
  };

  /**
   * Resync from Sonarr/Radarr
   */
  const handleResyncFromArr = async () => {
    try {
      setProgress({ type: 'resync', file: 'external services', progress: 0 });
      const response = await apiFetch('/api/library/resync', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ path: currentPath }),
      });

      if (response.ok) {
        await loadCurrentDirectory();
      } else {
        setError('Failed to resync from external services');
      }
    } catch (error) {
      console.error('Failed to resync from external services:', error);
      setError('Failed to resync from external services');
    } finally {
      setProgress(null);
    }
  };

  /**
   * Tick or untick one of a media file's subtitles.
   *
   * Order is preserved rather than sorted: the first subtitle picked is sent as
   * the primary and renders on top of each stacked cue, so re-ordering the
   * selection would silently change the output.
   *
   * @param {string} filePath - Path of the media file the subtitle belongs to.
   * @param {string} subPath - Path of the subtitle being toggled.
   */
  const toggleSubtitleSelection = (filePath, subPath) => {
    setSubtitleSelection(prev => {
      const current = prev[filePath] || [];
      const next = current.includes(subPath)
        ? current.filter(p => p !== subPath)
        : [...current, subPath];
      return { ...prev, [filePath]: next };
    });
  };

  /**
   * Combine the two selected subtitles into one bilingual file.
   *
   * Both languages must already exist on disk — this does not translate, so no
   * translation service is involved. The server picks the sentinel-language
   * output name so media servers treat the result as a distinct track.
   *
   * @param {Object} item - The media file whose subtitles are selected.
   */
  const combineSubtitles = async item => {
    const selected = subtitleSelection[item.path] || [];
    if (selected.length !== 2) return;

    setCombineError(null);
    setCombineResult(null);
    try {
      const response = await apiFetch('/api/subtitles/stack', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ primary: selected[0], secondary: selected[1] }),
      });
      if (response.ok) {
        const data = await response.json();
        setCombineResult(`Combined into ${data.output}`);
        setSubtitleSelection(prev => ({ ...prev, [item.path]: [] }));
        await loadCurrentDirectory();
      } else {
        // A silent failure here would look exactly like success, which is the
        // recurring defect elsewhere in this UI.
        const detail = await response.text();
        setCombineError(detail.trim() || `Combine failed (${response.status})`);
      }
    } catch (error) {
      setCombineError(error.message);
    }
  };

  /**
   * Navigate to a subdirectory
   */
  const navigateToPath = path => {
    setCurrentPath(path);
    setSelectedFiles(new Set());
  };

  /**
   * Get breadcrumb items from current path
   */
  const getBreadcrumbs = () => {
    const parts = currentPath.split('/').filter(Boolean);
    const breadcrumbs = [{ name: 'Root', path: '/' }];

    let currentBreadcrumbPath = '';
    parts.forEach(part => {
      currentBreadcrumbPath += '/' + part;
      breadcrumbs.push({ name: part, path: currentBreadcrumbPath });
    });

    return breadcrumbs;
  };

  /**
   * Get file type icon
   */
  const getFileIcon = item => {
    if (item.type === 'directory') return <FolderIcon />;
    if (item.isVideo) return item.isTvShow ? <TvIcon /> : <MovieIcon />;
    if (item.isSubtitle) return <SubtitleIcon />;
    return <InfoIcon />;
  };

  /**
   * Handle file action menu
   */
  const handleActionMenu = (event, file) => {
    event.stopPropagation();
    setActionMenu({ anchor: event.currentTarget, file });
  };

  const closeActionMenu = () => {
    setActionMenu({ anchor: null, file: null });
  };

  /**
   * Handle file operations
   */
  const handleOperation = async (type, file) => {
    closeActionMenu();
    setOperationDialog({ open: true, type, file });
  };

  const executeOperation = async () => {
    const { type, file } = operationDialog;
    if (!file || !type) return;

    setProgress({ type, file: file.name, progress: 0 });

    try {
      let endpoint = '';
      let body = {};

      switch (type) {
        case 'extract':
          endpoint = '/api/extract';
          body = { path: file.path };
          break;
        case 'search':
          endpoint = '/api/search';
          body = { path: file.path, language: 'en' };
          break;
        case 'translate':
          endpoint = '/api/translate';
          body = { path: file.path, targetLanguage: 'es' };
          break;
        default:
          return;
      }

      const response = await fetch(endpoint, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(body),
      });

      if (response.ok) {
        await loadCurrentDirectory(); // Refresh to show new files
      }
    } catch (error) {
      console.error(`Failed to ${type}:`, error);
    } finally {
      setProgress(null);
      setOperationDialog({ open: false, type: null, file: null });
    }
  };

  /**
   * Assign (or clear) a language profile across every selected file.
   *
   * This replaces a previous handleBulkOperation that posted to
   * /api/bulk-operation. That endpoint was never mounted, and nothing in the
   * component ever called the function — there was no selection UI to trigger
   * it — so there was no request contract to preserve. Its errors also went to
   * console.error only, which is precisely how a bulk action fails invisibly.
   *
   * An empty profileId clears the assignment, matching the endpoint's contract.
   */
  const handleBulkProfileAssign = async profileId => {
    if (selectedFiles.size === 0) return;

    const mediaIds = Array.from(selectedFiles);
    setProgress({
      type: 'profile',
      file: `${mediaIds.length} files`,
      progress: 0,
    });
    setBulkResult(null);

    try {
      const response = await apiFetch('/api/media/profiles/bulk', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ profile_id: profileId, media_ids: mediaIds }),
      });

      // A non-OK status here is a whole-request failure (unknown profile,
      // malformed body). Per-item failures come back inside a 200.
      if (!response.ok) {
        const detail = await response.text();
        setError(
          `Failed to assign profile: ${detail.trim() || response.status}`
        );
        return;
      }

      const data = await response.json();
      setBulkResult(data);
      // Only clear the selection when everything landed. Leaving the failed
      // items selected lets the user retry them without reselecting.
      if (!data.failed) {
        setSelectedFiles(new Set());
      } else {
        const failedIds = new Set(
          (data.results || []).filter(r => !r.ok).map(r => r.media_id)
        );
        setSelectedFiles(failedIds);
      }
      await loadCurrentDirectory();
    } catch (error) {
      console.error('Failed bulk profile assign:', error);
      setError('Failed to assign profile');
    } finally {
      setProgress(null);
    }
  };

  /**
   * Toggle file selection for bulk operations
   */
  const toggleFileSelection = filePath => {
    const newSelection = new Set(selectedFiles);
    if (newSelection.has(filePath)) {
      newSelection.delete(filePath);
    } else {
      newSelection.add(filePath);
    }
    setSelectedFiles(newSelection);
  };

  if (loading && items.length === 0) {
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
      {/* Backend availability warning */}
      {!backendAvailable && (
        <Alert severity="error" sx={{ mb: 3 }}>
          Backend service is not available. Media library browsing and subtitle
          management features are currently disabled.
        </Alert>
      )}

      {/* Error display */}
      {error && (
        <Alert severity="error" sx={{ mb: 3 }} onClose={() => setError(null)}>
          {error}
        </Alert>
      )}

      {combineError && (
        <Alert
          severity="error"
          sx={{ mb: 3 }}
          onClose={() => setCombineError(null)}
        >
          {combineError}
        </Alert>
      )}

      {combineResult && (
        <Alert
          severity="success"
          sx={{ mb: 3 }}
          onClose={() => setCombineResult(null)}
        >
          {combineResult}
        </Alert>
      )}

      {/* Header */}
      <Box
        display="flex"
        justifyContent="space-between"
        alignItems="center"
        mb={3}
      >
        <Typography variant="h4" component="h1">
          Media Library
        </Typography>
        <Box display="flex" alignItems="center" gap={1}>
          {/* Library Management */}
          <Button
            variant="contained"
            startIcon={<AddIcon />}
            onClick={() => setAddLibraryDialog(true)}
            disabled={!backendAvailable}
            size="small"
          >
            Add Library Path
          </Button>

          {/* Page-specific Actions */}
          {currentPath !== '/' && (
            <>
              <Button
                variant="outlined"
                startIcon={<StorageIcon />}
                onClick={handleRescanDisk}
                disabled={!backendAvailable}
                size="small"
              >
                Rescan Disk
              </Button>
              <Button
                variant="outlined"
                startIcon={<SyncIcon />}
                onClick={handleResyncFromSonarr}
                disabled={!backendAvailable}
                size="small"
              >
                Resync from Sonarr/Radarr
              </Button>
            </>
          )}

          {/*
            Mass edit is a mode rather than always-on checkboxes: selection
            controls on every row make ordinary browsing noisier, and the
            enclosing Card is itself a navigation target.
          */}
          <Button
            variant={bulkMode ? 'contained' : 'outlined'}
            size="small"
            onClick={() => {
              setBulkMode(!bulkMode);
              setSelectedFiles(new Set());
              setBulkResult(null);
            }}
            disabled={!backendAvailable}
          >
            {bulkMode ? 'Exit Mass Edit' : 'Mass Edit'}
          </Button>

          {/* View mode toggle */}
          <ToggleButtonGroup
            value={viewMode}
            exclusive
            onChange={(e, newValue) => newValue && setViewMode(newValue)}
            size="small"
          >
            <ToggleButton value="list">
              <ListIcon />
            </ToggleButton>
            <ToggleButton value="grid">
              <GridViewIcon />
            </ToggleButton>
          </ToggleButtonGroup>
        </Box>
      </Box>

      {/* Mass-edit toolbar */}
      {bulkMode && (
        <Paper sx={{ p: 2, mb: 2 }} variant="outlined">
          <Box
            display="flex"
            alignItems="center"
            gap={2}
            flexWrap="wrap"
            data-testid="mass-edit-toolbar"
          >
            <Typography variant="body2">
              {selectedFiles.size} selected
            </Typography>

            <FormControl size="small" sx={{ minWidth: 220 }}>
              <InputLabel id="bulk-profile-label">Language profile</InputLabel>
              <Select
                labelId="bulk-profile-label"
                label="Language profile"
                value={bulkProfileId}
                onChange={event => setBulkProfileId(event.target.value)}
              >
                {/*
                  The empty value is a real choice, not a placeholder: the
                  endpoint treats an empty profile_id as "clear the
                  assignment", which is how a file is returned to the scan's
                  own language.
                */}
                <MenuItem value="">
                  <em>None (clear assignment)</em>
                </MenuItem>
                {languageProfiles.map(profile => (
                  <MenuItem key={profile.id} value={profile.id}>
                    {profile.name}
                    {profile.is_default ? ' (default)' : ''}
                  </MenuItem>
                ))}
              </Select>
            </FormControl>

            <Button
              variant="contained"
              size="small"
              disabled={selectedFiles.size === 0 || !!progress}
              onClick={() => handleBulkProfileAssign(bulkProfileId)}
            >
              Apply to {selectedFiles.size}
            </Button>

            <Button
              size="small"
              disabled={selectedFiles.size === 0}
              onClick={() => setSelectedFiles(new Set())}
            >
              Clear selection
            </Button>

            <Button
              size="small"
              onClick={() =>
                setSelectedFiles(
                  new Set(
                    // getTabContent() returns null on the Library Paths tab,
                    // which manages paths rather than listing media. The
                    // toolbar is shown whenever mass edit is on, independent of
                    // the active tab, so that tab is reachable with this button
                    // on screen and an unguarded .filter throws.
                    (getTabContent() || [])
                      .filter(item => isMediaFile(item) && item.path)
                      .map(item => item.path)
                  )
                )
              }
            >
              Select all files
            </Button>
          </Box>

          {bulkResult && (
            <Alert
              severity={bulkResult.failed ? 'warning' : 'success'}
              sx={{ mt: 2 }}
              onClose={() => setBulkResult(null)}
            >
              {bulkResult.failed
                ? `${bulkResult.succeeded} of ${
                    bulkResult.succeeded + bulkResult.failed
                  } assigned — ${bulkResult.failed} failed and remain selected.`
                : `${bulkResult.succeeded} ${
                    bulkResult.profile_id ? 'assigned' : 'cleared'
                  }.`}
            </Alert>
          )}
        </Paper>
      )}

      {/* Sonarr-style Navigation Tabs */}
      <Box sx={{ borderBottom: 1, borderColor: 'divider', mb: 3 }}>
        <Tabs
          value={activeTab}
          onChange={handleTabChange}
          variant="scrollable"
          scrollButtons="auto"
        >
          <Tab label="All Media" />
          <Tab label="Movies" />
          <Tab label="TV Shows" />
          <Tab label="Library Paths" />
        </Tabs>
      </Box>

      {/* Tab Content */}
      {activeTab === 3 ? (
        renderLibraryPathsTab()
      ) : (
        <>
          {/* Breadcrumb Navigation */}
          {currentPath !== '/' && (
            <Breadcrumbs sx={{ mb: 2 }}>
              <Link
                component="button"
                variant="body1"
                onClick={() => navigateToPath('/')}
                underline="hover"
              >
                Home
              </Link>
              {currentPath
                .split('/')
                .filter(Boolean)
                .map((segment, index, array) => {
                  const path = '/' + array.slice(0, index + 1).join('/');
                  const isLast = index === array.length - 1;

                  return isLast ? (
                    <Typography key={path} color="text.primary">
                      {segment}
                    </Typography>
                  ) : (
                    <Link
                      key={path}
                      component="button"
                      variant="body1"
                      onClick={() => navigateToPath(path)}
                      underline="hover"
                    >
                      {segment}
                    </Link>
                  );
                })}
            </Breadcrumbs>
          )}

          {/* Content based on tab */}
          {viewMode === 'grid' ? (
            <Grid container spacing={2}>
              {getTabContent()
                .filter(
                  item =>
                    item.isDirectory ||
                    item.name?.match(/\.(mp4|mkv|avi|mov|wmv|flv|webm|m4v)$/i)
                )
                .map(item => (
                  <GridItem key={item.path} item={item} />
                ))}
            </Grid>
          ) : (
            renderListView(getTabContent())
          )}
        </>
      )}

      {/* Add Library Path Dialog */}
      <Dialog
        open={addLibraryDialog}
        onClose={() => setAddLibraryDialog(false)}
        maxWidth="sm"
        fullWidth
      >
        <DialogTitle>Add Library Path</DialogTitle>
        <DialogContent>
          <Typography variant="body2" sx={{ mb: 2 }}>
            Add a new directory path to your media library. This path will be
            scanned for media files and subtitles.
          </Typography>
          <TextField
            autoFocus
            margin="dense"
            label="Library Path"
            fullWidth
            variant="outlined"
            value={newLibraryPath}
            onChange={e => setNewLibraryPath(e.target.value)}
            placeholder="/path/to/your/media/folder"
            helperText="Enter the full path to a directory containing your media files"
          />
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setAddLibraryDialog(false)}>Cancel</Button>
          <Button
            onClick={handleAddLibraryPath}
            variant="contained"
            disabled={!newLibraryPath.trim()}
          >
            Add Path
          </Button>
        </DialogActions>
      </Dialog>
    </Box>
  );
}

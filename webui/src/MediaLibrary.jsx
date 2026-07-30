// file: webui/src/MediaLibrary.jsx
// version: 2.0.0
// guid: 1a2b3c4d-5e6f-7a8b-9c0d-1e2f3a4b5c6d
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
  CircularProgress,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  Grid,
  IconButton,
  Link,
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
        setItems(data.items || []);
      } else {
        setItems([]);
      }
    } catch (error) {
      console.error('Failed to load directory:', error);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadCurrentDirectory();
    loadLibraryPaths();

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
          item => item.type === 'tv' || (!item.type && item.is_dir)
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
            item.is_dir ||
            item.name?.match(/\.(mp4|mkv|avi|mov|wmv|flv|webm|m4v)$/i)
        )
        .map(item => (
          <Grid item xs={12} key={item.path || Math.random()}>
            <Card
              sx={{
                cursor: item.is_dir ? 'pointer' : 'default',
                '&:hover': {
                  backgroundColor: 'action.hover',
                },
              }}
              onClick={() => {
                if (item.is_dir) {
                  setCurrentPath(item.path);
                  loadCurrentDirectory();
                } else {
                  navigate(
                    `/details?title=${encodeURIComponent(item.name || '')}&path=${encodeURIComponent(item.path || '')}`
                  );
                }
              }}
            >
              <CardContent>
                <Box display="flex" alignItems="center">
                  <Box sx={{ mr: 2 }}>
                    {item.is_dir ? (
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
                  {!item.is_dir && (
                    <IconButton size="small">
                      <MoreIcon />
                    </IconButton>
                  )}
                </Box>
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
   * Handle bulk operations
   */
  const handleBulkOperation = async type => {
    if (selectedFiles.size === 0) return;

    const files = Array.from(selectedFiles)
      .map(path => items.find(item => item?.path === path))
      .filter(Boolean);

    setProgress({ type, file: `${files.length} files`, progress: 0 });

    try {
      await apiFetch('/api/bulk-operation', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          type,
          files: files.map(f => f?.path).filter(Boolean),
          language: 'en',
        }),
      });

      await loadCurrentDirectory();
      setSelectedFiles(new Set());
    } catch (error) {
      console.error(`Failed bulk ${type}:`, error);
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
                onClick={() => loadDirectory('/')}
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
                      onClick={() => loadDirectory(path)}
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
                    item.is_dir ||
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

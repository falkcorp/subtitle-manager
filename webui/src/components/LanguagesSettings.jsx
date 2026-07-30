// file: webui/src/components/LanguagesSettings.jsx
// version: 1.0.0
// guid: 123e4567-e89b-12d3-a456-426614174000

import {
  Add as AddIcon,
  Delete as DeleteIcon,
  Edit as EditIcon,
  Star as DefaultIcon,
  Translate as LanguageIcon,
} from '@mui/icons-material';
import {
  Alert,
  Box,
  Button,
  Card,
  CardContent,
  CardHeader,
  Chip,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  Fab,
  FormControlLabel,
  Grid,
  IconButton,
  List,
  ListItem,
  ListItemText,
  Paper,
  Switch,
  TextField,
  Typography,
  Autocomplete,
  MenuItem,
  FormControl,
  InputLabel,
  Select,
} from '@mui/material';
import { useEffect, useState } from 'react';
import { apiFetch } from '../services/api.js';

// Common language options
const COMMON_LANGUAGES = [
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
  { code: 'ar', name: 'Arabic' },
  { code: 'hi', name: 'Hindi' },
  { code: 'th', name: 'Thai' },
  { code: 'tr', name: 'Turkish' },
  { code: 'pl', name: 'Polish' },
  { code: 'nl', name: 'Dutch' },
  { code: 'sv', name: 'Swedish' },
  { code: 'da', name: 'Danish' },
  { code: 'no', name: 'Norwegian' },
  { code: 'fi', name: 'Finnish' },
];

/**
 * Language Profiles Settings Component
 * Manages language profiles similar to Bazarr for subtitle preferences
 */
export default function LanguagesSettings({ backendAvailable = true }) {
  const [profiles, setProfiles] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);
  const [openDialog, setOpenDialog] = useState(false);
  const [editingProfile, setEditingProfile] = useState(null);
  const [formData, setFormData] = useState({
    name: '',
    languages: [],
    cutoff_score: 80,
    is_default: false,
  });

  // Load profiles on component mount
  useEffect(() => {
    if (backendAvailable) {
      loadProfiles();
    }
  }, [backendAvailable]);

  const loadProfiles = async () => {
    try {
      setLoading(true);
      const response = await apiFetch('/api/language-profiles');
      if (!response.ok) {
        throw new Error(`Failed to load profiles: ${response.statusText}`);
      }
      const data = await response.json();
      setProfiles(data || []);
    } catch (err) {
      console.error('Error loading language profiles:', err);
      setError(`Failed to load language profiles: ${err.message}`);
    } finally {
      setLoading(false);
    }
  };

  const handleCreateProfile = () => {
    setEditingProfile(null);
    setFormData({
      name: '',
      languages: [],
      cutoff_score: 80,
      is_default: false,
    });
    setOpenDialog(true);
  };

  const handleEditProfile = profile => {
    setEditingProfile(profile);
    setFormData({
      name: profile.name || '',
      languages: profile.languages || [],
      cutoff_score: profile.cutoff_score || 80,
      is_default: profile.is_default || false,
    });
    setOpenDialog(true);
  };

  const handleDeleteProfile = async profileId => {
    if (!confirm('Are you sure you want to delete this language profile?')) {
      return;
    }

    try {
      const response = await apiFetch(`/api/language-profiles/${profileId}`, {
        method: 'DELETE',
      });
      if (!response.ok) {
        throw new Error(`Failed to delete profile: ${response.statusText}`);
      }
      await loadProfiles();
    } catch (err) {
      console.error('Error deleting profile:', err);
      setError(`Failed to delete profile: ${err.message}`);
    }
  };

  const handleSaveProfile = async () => {
    try {
      const profileData = {
        ...formData,
        languages: formData.languages.map((lang, index) => ({
          language: lang.language,
          priority: index + 1,
          forced: lang.forced || false,
          hi: lang.hi || false,
        })),
      };

      const isEditing = !!editingProfile;
      const url = isEditing
        ? `/api/language-profiles/${editingProfile.id}`
        : '/api/language-profiles';
      const method = isEditing ? 'PUT' : 'POST';

      const response = await fetch(url, {
        method,
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(profileData),
      });

      if (!response.ok) {
        const errorText = await response.text();
        throw new Error(`Failed to save profile: ${errorText}`);
      }

      setOpenDialog(false);
      await loadProfiles();
    } catch (err) {
      console.error('Error saving profile:', err);
      setError(`Failed to save profile: ${err.message}`);
    }
  };

  const handleLanguageAdd = () => {
    setFormData({
      ...formData,
      languages: [
        ...formData.languages,
        {
          language: '',
          priority: formData.languages.length + 1,
          forced: false,
          hi: false,
        },
      ],
    });
  };

  const handleLanguageRemove = index => {
    const newLanguages = formData.languages.filter((_, i) => i !== index);
    setFormData({
      ...formData,
      languages: newLanguages,
    });
  };

  const handleLanguageChange = (index, field, value) => {
    const newLanguages = [...formData.languages];
    newLanguages[index] = {
      ...newLanguages[index],
      [field]: value,
    };
    setFormData({
      ...formData,
      languages: newLanguages,
    });
  };

  const getLanguageName = code => {
    const lang = COMMON_LANGUAGES.find(l => l.code === code);
    return lang ? lang.name : code;
  };

  if (!backendAvailable) {
    return (
      <Alert severity="warning">
        Backend service is not available. Language profiles cannot be managed.
      </Alert>
    );
  }

  if (loading) {
    return (
      <Box display="flex" justifyContent="center" p={3}>
        <Typography>Loading language profiles...</Typography>
      </Box>
    );
  }

  return (
    <Box>
      <Box
        display="flex"
        justifyContent="space-between"
        alignItems="center"
        mb={3}
      >
        <Typography variant="h6" component="h2">
          Language Profiles
        </Typography>
        <Button
          variant="contained"
          startIcon={<AddIcon />}
          onClick={handleCreateProfile}
        >
          Add Profile
        </Button>
      </Box>

      {error && (
        <Alert severity="error" sx={{ mb: 2 }} onClose={() => setError(null)}>
          {error}
        </Alert>
      )}

      <Grid container spacing={2}>
        {profiles.map(profile => (
          <Grid item xs={12} md={6} key={profile.id}>
            <Card>
              <CardHeader
                title={
                  <Box display="flex" alignItems="center" gap={1}>
                    <Typography variant="h6">{profile.name}</Typography>
                    {profile.is_default && (
                      <Chip
                        icon={<DefaultIcon />}
                        label="Default"
                        size="small"
                        color="primary"
                      />
                    )}
                  </Box>
                }
                action={
                  <Box>
                    <IconButton
                      onClick={() => handleEditProfile(profile)}
                      size="small"
                    >
                      <EditIcon />
                    </IconButton>
                    <IconButton
                      onClick={() => handleDeleteProfile(profile.id)}
                      size="small"
                      color="error"
                    >
                      <DeleteIcon />
                    </IconButton>
                  </Box>
                }
              />
              <CardContent>
                <Typography variant="body2" color="textSecondary" gutterBottom>
                  Cutoff Score: {profile.cutoff_score}
                </Typography>
                <Box>
                  <Typography variant="subtitle2" gutterBottom>
                    Languages:
                  </Typography>
                  <Box display="flex" flexWrap="wrap" gap={1}>
                    {profile.languages?.map((lang, index) => (
                      <Chip
                        key={index}
                        size="small"
                        label={`${getLanguageName(lang.language)} (${lang.priority})`}
                        variant={lang.forced ? 'filled' : 'outlined'}
                        color={lang.hi ? 'secondary' : 'default'}
                      />
                    ))}
                  </Box>
                </Box>
              </CardContent>
            </Card>
          </Grid>
        ))}
      </Grid>

      {/* Profile Editor Dialog */}
      <Dialog
        open={openDialog}
        onClose={() => setOpenDialog(false)}
        maxWidth="md"
        fullWidth
      >
        <DialogTitle>
          {editingProfile ? 'Edit Language Profile' : 'Create Language Profile'}
        </DialogTitle>
        <DialogContent>
          <Box sx={{ display: 'flex', flexDirection: 'column', gap: 2, pt: 1 }}>
            <TextField
              fullWidth
              label="Profile Name"
              value={formData.name}
              onChange={e => setFormData({ ...formData, name: e.target.value })}
              placeholder="e.g., Movies, TV Shows, Anime"
            />

            <TextField
              fullWidth
              type="number"
              label="Cutoff Score"
              value={formData.cutoff_score}
              onChange={e =>
                setFormData({
                  ...formData,
                  cutoff_score: parseInt(e.target.value) || 80,
                })
              }
              helperText="Quality threshold for subtitle downloads (0-100)"
              inputProps={{ min: 0, max: 100 }}
            />

            <FormControlLabel
              control={
                <Switch
                  checked={formData.is_default}
                  onChange={e =>
                    setFormData({ ...formData, is_default: e.target.checked })
                  }
                />
              }
              label="Set as Default Profile"
            />

            <Box>
              <Box
                display="flex"
                justifyContent="space-between"
                alignItems="center"
                mb={2}
              >
                <Typography variant="subtitle1">Languages</Typography>
                <Button
                  variant="outlined"
                  size="small"
                  startIcon={<AddIcon />}
                  onClick={handleLanguageAdd}
                >
                  Add Language
                </Button>
              </Box>

              {formData.languages.map((lang, index) => (
                <Box
                  key={index}
                  sx={{ display: 'flex', gap: 2, mb: 2, alignItems: 'center' }}
                >
                  <Autocomplete
                    options={COMMON_LANGUAGES}
                    getOptionLabel={option => `${option.name} (${option.code})`}
                    value={
                      COMMON_LANGUAGES.find(l => l.code === lang.language) ||
                      null
                    }
                    onChange={(_, newValue) =>
                      handleLanguageChange(
                        index,
                        'language',
                        newValue?.code || ''
                      )
                    }
                    renderInput={params => (
                      <TextField {...params} label="Language" size="small" />
                    )}
                    sx={{ minWidth: 200 }}
                  />

                  <FormControlLabel
                    control={
                      <Switch
                        checked={lang.forced || false}
                        onChange={e =>
                          handleLanguageChange(
                            index,
                            'forced',
                            e.target.checked
                          )
                        }
                        size="small"
                      />
                    }
                    label="Forced"
                  />

                  <FormControlLabel
                    control={
                      <Switch
                        checked={lang.hi || false}
                        onChange={e =>
                          handleLanguageChange(index, 'hi', e.target.checked)
                        }
                        size="small"
                      />
                    }
                    label="HI"
                  />

                  <IconButton
                    onClick={() => handleLanguageRemove(index)}
                    size="small"
                    color="error"
                  >
                    <DeleteIcon />
                  </IconButton>
                </Box>
              ))}

              {formData.languages.length === 0 && (
                <Typography
                  variant="body2"
                  color="textSecondary"
                  sx={{ textAlign: 'center', py: 2 }}
                >
                  No languages added. Click "Add Language" to get started.
                </Typography>
              )}
            </Box>
          </Box>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setOpenDialog(false)}>Cancel</Button>
          <Button
            onClick={handleSaveProfile}
            variant="contained"
            disabled={!formData.name || formData.languages.length === 0}
          >
            {editingProfile ? 'Update' : 'Create'} Profile
          </Button>
        </DialogActions>
      </Dialog>
    </Box>
  );
}

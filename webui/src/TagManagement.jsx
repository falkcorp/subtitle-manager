// file: webui/src/TagManagement.jsx
import {
  Alert,
  Box,
  Button,
  CircularProgress,
  Paper,
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableRow,
  TextField,
  Typography,
} from '@mui/material';
import { useCallback, useEffect, useState } from 'react';
import { apiFetch } from './services/api.js';

/**
 * TagManagement displays existing tags and allows creation/deletion.
 *
 * @param {Object} props - Component props
 * @param {boolean} props.backendAvailable - Whether the backend is reachable
 */
export default function TagManagement({ backendAvailable = true }) {
  const [tags, setTags] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);
  const [newTag, setNewTag] = useState('');
  const [editId, setEditId] = useState(null);
  const [editName, setEditName] = useState('');
  const [originalEditName, setOriginalEditName] = useState('');

  const loadTags = useCallback(async () => {
    if (!backendAvailable) {
      setLoading(false);
      return;
    }
    try {
      setLoading(true);
      setError(null);
      const res = await apiFetch('/api/tags');
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      const data = await res.json();
      setTags(Array.isArray(data) ? data : []);
    } catch {
      setError('Failed to fetch tag list');
    } finally {
      setLoading(false);
    }
  }, [backendAvailable]);

  const addTag = async () => {
    if (!newTag) return;
    try {
      const res = await apiFetch('/api/tags', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ name: newTag }),
      });
      if (res.ok) {
        setNewTag('');
        loadTags();
      } else {
        setError(`Failed to add tag: HTTP ${res.status}`);
      }
    } catch {
      setError('Failed to add tag');
    }
  };

  const startEdit = tag => {
    setEditId(tag.id);
    setEditName(tag.name);
    setOriginalEditName(tag.name);
  };

  const saveEdit = async () => {
    try {
      const res = await apiFetch(`/api/tags/${editId}`, {
        method: 'PATCH',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ name: editName }),
      });
      if (res.ok) {
        setEditId(null);
        setEditName('');
        setOriginalEditName('');
        loadTags();
      } else {
        setError(`Failed to update tag: HTTP ${res.status}`);
      }
    } catch {
      setError('Failed to update tag');
    }
  };

  const cancelEdit = () => {
    setEditId(null);
    setEditName('');
    setOriginalEditName('');
  };

  const deleteTag = async id => {
    if (!window.confirm('Delete this tag?')) return;
    try {
      const res = await apiFetch(`/api/tags/${id}`, { method: 'DELETE' });
      if (res.ok) {
        loadTags();
      } else {
        setError(`Failed to delete tag: HTTP ${res.status}`);
      }
    } catch {
      setError('Failed to delete tag');
    }
  };

  useEffect(() => {
    loadTags();
  }, [loadTags]);

  if (loading) {
    return (
      <Box display="flex" justifyContent="center" alignItems="center" p={4}>
        <CircularProgress />
      </Box>
    );
  }

  return (
    <Box>
      <Typography variant="h6" gutterBottom>
        Tag Management
      </Typography>
      {!backendAvailable && (
        <Alert severity="error" sx={{ mb: 3 }}>
          Backend service is not available.
        </Alert>
      )}
      {error && (
        <Alert severity="error" sx={{ mb: 3 }} onClose={() => setError(null)}>
          {error}
        </Alert>
      )}
      <Box display="flex" gap={2} mb={2}>
        <TextField
          label="New Tag"
          value={newTag}
          onChange={e => setNewTag(e.target.value)}
          size="small"
        />
        <Button variant="contained" onClick={addTag} disabled={!newTag}>
          Add
        </Button>
      </Box>
      <Paper variant="outlined">
        <Table size="small">
          <TableHead>
            <TableRow>
              <TableCell>Name</TableCell>
              <TableCell>Actions</TableCell>
            </TableRow>
          </TableHead>
          <TableBody>
            {tags.map(tag => (
              <TableRow key={tag.id}>
                <TableCell>
                  {editId === tag.id ? (
                    <TextField
                      value={editName}
                      onChange={e => setEditName(e.target.value)}
                      size="small"
                    />
                  ) : (
                    tag.name
                  )}
                </TableCell>
                <TableCell>
                  {editId === tag.id ? (
                    <>
                      <Button
                        size="small"
                        onClick={saveEdit}
                        disabled={
                          !editName.trim() || editName === originalEditName
                        }
                      >
                        Save
                      </Button>
                      <Button size="small" onClick={cancelEdit}>
                        Cancel
                      </Button>
                    </>
                  ) : (
                    <>
                      <Button size="small" onClick={() => startEdit(tag)}>
                        Edit
                      </Button>
                      <Button size="small" onClick={() => deleteTag(tag.id)}>
                        Delete
                      </Button>
                    </>
                  )}
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </Paper>
    </Box>
  );
}

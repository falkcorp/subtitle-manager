// file: webui/src/components/TagSelector.jsx
import {
  Box,
  Button,
  Chip,
  Checkbox,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  List,
  ListItem,
  ListItemText,
} from '@mui/material';
import { useCallback, useEffect, useState } from 'react';
import { apiFetch } from '../services/api.js';

/**
 * TagSelector allows assigning tags to a media item identified by path.
 *
 * @param {Object} props - Component props
 * @param {string} props.path - Media file path
 */
export default function TagSelector({ path }) {
  const [allTags, setAllTags] = useState([]);
  const [itemTags, setItemTags] = useState([]);
  const [open, setOpen] = useState(false);

  const loadTags = useCallback(async () => {
    if (!path) return;
    try {
      const res = await apiFetch(
        `/api/library/tags?path=${encodeURIComponent(path)}`
      );
      if (res.ok) {
        const data = await res.json();
        setItemTags(Array.isArray(data) ? data : []);
      }
    } catch (error) {
      console.error('Failed to load tags for path:', path, error);
    }
  }, [path]);

  const loadAll = useCallback(async () => {
    try {
      const res = await apiFetch('/api/tags');
      if (res.ok) {
        const data = await res.json();
        setAllTags(Array.isArray(data) ? data : []);
      }
    } catch (error) {
      console.error('Failed to fetch all tags:', error);
    }
  }, []);

  useEffect(() => {
    loadTags();
  }, [loadTags]);

  const toggleTag = async (tagId, checked) => {
    try {
      await apiFetch('/api/library/tags', {
        method: checked ? 'POST' : 'DELETE',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ path, tag_id: tagId }),
      });
      loadTags();
    } catch {
      // ignore
    }
  };

  const assigned = id => itemTags.some(t => t.id === String(id));

  return (
    <Box>
      <Box display="flex" flexWrap="wrap" gap={1} mb={1}>
        {itemTags.map(t => (
          <Chip key={t.id} label={t.name} size="small" />
        ))}
        <Button
          size="small"
          onClick={() => {
            setOpen(true);
            loadAll();
          }}
        >
          Manage Tags
        </Button>
      </Box>
      <Dialog
        open={open}
        onClose={() => setOpen(false)}
        maxWidth="xs"
        fullWidth
      >
        <DialogTitle>Manage Tags</DialogTitle>
        <DialogContent>
          <List>
            {allTags.map(tag => (
              <ListItem key={tag.id} disableGutters>
                <Checkbox
                  edge="start"
                  checked={assigned(tag.id)}
                  onChange={e => toggleTag(tag.id, e.target.checked)}
                />
                <ListItemText primary={tag.name} />
              </ListItem>
            ))}
          </List>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setOpen(false)}>Close</Button>
        </DialogActions>
      </Dialog>
    </Box>
  );
}

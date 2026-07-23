import {
  CheckCircle as OkIcon,
  Timeline as DriftIcon,
  Warning as WarnIcon,
} from '@mui/icons-material';
import {
  Alert,
  Box,
  Button,
  Card,
  CardContent,
  Grid,
  LinearProgress,
  TextField,
  Typography,
} from '@mui/material';
import { useState } from 'react';

import { apiService } from './services/api.js';

/**
 * Verify runs subtitle drift verification against a media file's audio and
 * reports whether the subtitle is in sync, has a constant offset, or drifts
 * (speeds up / slows down). It calls the /api/verify endpoint, which samples
 * audio windows and transcribes them with Whisper, so it can take a while.
 *
 * @param {Object} props - Component props
 * @param {boolean} props.backendAvailable - Whether the backend is reachable
 */
export default function Verify({ backendAvailable = true }) {
  const [media, setMedia] = useState('');
  const [subtitle, setSubtitle] = useState('');
  const [lang, setLang] = useState('en');
  const [running, setRunning] = useState(false);
  const [error, setError] = useState('');
  const [report, setReport] = useState(null);

  const handleVerify = async () => {
    setError('');
    setReport(null);
    if (!media || !subtitle || !lang) {
      setError('Media path, subtitle path, and language are all required.');
      return;
    }
    setRunning(true);
    try {
      const res = await apiService.post('/api/verify', {
        media,
        subtitle,
        lang,
      });
      // apiService.post returns the parsed JSON body (or a Response); handle both.
      const data = res && res.json ? await res.json() : res;
      setReport(data);
    } catch (e) {
      setError(e?.message || 'Verification failed.');
    } finally {
      setRunning(false);
    }
  };

  const ms = v => (v == null ? '0' : Math.round(v).toString());

  return (
    <Box sx={{ p: 3 }}>
      <Typography variant="h4" gutterBottom>
        Verify Subtitle Sync
      </Typography>
      <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>
        Checks a subtitle against the media audio at several timestamps and
        reports a constant offset or speed-up/slow-down drift (e.g. a 23.976 vs
        25 fps mismatch). Uses Whisper, so it can take a minute or two.
      </Typography>

      <Card sx={{ mb: 3 }}>
        <CardContent>
          <Grid container spacing={2}>
            <Grid item xs={12}>
              <TextField
                fullWidth
                label="Media file path"
                value={media}
                onChange={e => setMedia(e.target.value)}
                placeholder="/media/movies/Example (2020)/Example.mkv"
              />
            </Grid>
            <Grid item xs={12} sm={8}>
              <TextField
                fullWidth
                label="Subtitle file path"
                value={subtitle}
                onChange={e => setSubtitle(e.target.value)}
                placeholder="/media/movies/Example (2020)/Example.en.srt"
              />
            </Grid>
            <Grid item xs={12} sm={4}>
              <TextField
                fullWidth
                label="Language"
                value={lang}
                onChange={e => setLang(e.target.value)}
                placeholder="en"
              />
            </Grid>
            <Grid item xs={12}>
              <Button
                variant="contained"
                onClick={handleVerify}
                disabled={!backendAvailable || running}
              >
                {running ? 'Verifying…' : 'Verify'}
              </Button>
            </Grid>
          </Grid>
          {running && <LinearProgress sx={{ mt: 2 }} />}
        </CardContent>
      </Card>

      {error && (
        <Alert severity="error" sx={{ mb: 2 }}>
          {error}
        </Alert>
      )}

      {report && (
        <Card>
          <CardContent>
            {report.InSync ? (
              <Alert icon={<OkIcon />} severity="success">
                In sync — no significant offset or drift detected.
              </Alert>
            ) : report.RateDrift ? (
              <Alert icon={<DriftIcon />} severity="warning">
                Speed drift detected: {report.SlopeMsPerSec?.toFixed(2)} ms/s
                {report.LikelyCause ? ` — likely ${report.LikelyCause}` : ''}.
              </Alert>
            ) : (
              <Alert icon={<WarnIcon />} severity="warning">
                Constant offset of {ms(report.InterceptMs)} ms — a simple shift
                would fix it.
              </Alert>
            )}
            <Box sx={{ mt: 2 }}>
              <Typography variant="body2">
                Anchors used: {report.Used} / {report.Anchors?.length ?? 0}
              </Typography>
              <Typography variant="body2">
                Offset at start: {ms(report.InterceptMs)} ms · drift{' '}
                {report.SlopeMsPerSec?.toFixed(2) ?? '0'} ms/s
              </Typography>
              <Typography variant="body2">
                Fit RMSE: {ms(report.RMSEMs)} ms (max residual{' '}
                {ms(report.MaxResidualMs)} ms)
              </Typography>
            </Box>
          </CardContent>
        </Card>
      )}
    </Box>
  );
}

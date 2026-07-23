// file: pkg/video/video.go
// version: 1.1.0
// guid: 8a9b0c1d-2e3f-4a5b-6c7d-8e9f0a1b2c3d

// Package video provides video analysis and processing utilities using ffmpeg and ffprobe.
// It supports extracting video metadata, analyzing video properties, and performing basic video operations.
package video

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

var ffprobePath = "ffprobe"

// SetFFprobePath allows overriding the default ffprobe binary path.
func SetFFprobePath(path string) {
	ffprobePath = path
}

// VideoInfo represents comprehensive video file information
type VideoInfo struct {
	Duration   time.Duration `json:"duration"`
	Width      int           `json:"width"`
	Height     int           `json:"height"`
	Bitrate    int64         `json:"bitrate"`
	FrameRate  float64       `json:"frame_rate"`
	Codec      string        `json:"codec"`
	Format     string        `json:"format"`
	FileSize   int64         `json:"file_size"`
	AudioCodec string        `json:"audio_codec"`
	AudioRate  int           `json:"audio_rate"`
}

// ffprobeResult represents the JSON output from ffprobe
type ffprobeResult struct {
	Format struct {
		Filename   string `json:"filename"`
		FormatName string `json:"format_name"`
		Duration   string `json:"duration"`
		Size       string `json:"size"`
		BitRate    string `json:"bit_rate"`
	} `json:"format"`
	Streams []struct {
		Index        int    `json:"index"`
		CodecType    string `json:"codec_type"`
		CodecName    string `json:"codec_name"`
		Width        int    `json:"width"`
		Height       int    `json:"height"`
		AvgFrameRate string `json:"avg_frame_rate"`
		Duration     string `json:"duration"`
		BitRate      string `json:"bit_rate"`
		SampleRate   string `json:"sample_rate"`
		Channels     int    `json:"channels"`
	} `json:"streams"`
}

// AnalyzeVideo extracts comprehensive metadata from a video file using ffprobe
func AnalyzeVideo(videoPath string) (*VideoInfo, error) {
	cmd := exec.CommandContext(context.Background(), ffprobePath,
		"-v", "quiet",
		"-print_format", "json",
		"-show_format",
		"-show_streams",
		videoPath)

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("ffprobe analysis failed: %w", err)
	}

	var result ffprobeResult
	if err := json.Unmarshal(output, &result); err != nil {
		return nil, fmt.Errorf("failed to parse ffprobe output: %w", err)
	}

	info := &VideoInfo{}

	// Parse format information
	if result.Format.Duration != "" {
		if duration, err := strconv.ParseFloat(result.Format.Duration, 64); err == nil {
			info.Duration = time.Duration(duration * float64(time.Second))
		}
	}

	if result.Format.Size != "" {
		if size, err := strconv.ParseInt(result.Format.Size, 10, 64); err == nil {
			info.FileSize = size
		}
	}

	if result.Format.BitRate != "" {
		if bitrate, err := strconv.ParseInt(result.Format.BitRate, 10, 64); err == nil {
			info.Bitrate = bitrate
		}
	}

	info.Format = result.Format.FormatName

	// Find video and audio streams
	for _, stream := range result.Streams {
		if stream.CodecType == "video" && info.Codec == "" {
			info.Codec = stream.CodecName
			info.Width = stream.Width
			info.Height = stream.Height

			// Parse frame rate
			if stream.AvgFrameRate != "" && stream.AvgFrameRate != "0/0" {
				parts := strings.Split(stream.AvgFrameRate, "/")
				if len(parts) == 2 {
					if num, err := strconv.ParseFloat(parts[0], 64); err == nil {
						if denom, err := strconv.ParseFloat(parts[1], 64); err == nil && denom != 0 {
							info.FrameRate = num / denom
						}
					}
				}
			}
		}

		if stream.CodecType == "audio" && info.AudioCodec == "" {
			info.AudioCodec = stream.CodecName
			if stream.SampleRate != "" {
				if rate, err := strconv.Atoi(stream.SampleRate); err == nil {
					info.AudioRate = rate
				}
			}
		}
	}

	return info, nil
}

// GetResolution returns a formatted resolution string (e.g., "1920x1080")
func (vi *VideoInfo) GetResolution() string {
	if vi.Width > 0 && vi.Height > 0 {
		return fmt.Sprintf("%dx%d", vi.Width, vi.Height)
	}
	return ""
}

// GetResolutionCategory returns common resolution categories (1080p, 720p, etc.)
func (vi *VideoInfo) GetResolutionCategory() string {
	if vi.Height >= 2160 {
		return "2160p"
	} else if vi.Height >= 1440 {
		return "1440p"
	} else if vi.Height >= 1080 {
		return "1080p"
	} else if vi.Height >= 720 {
		return "720p"
	} else if vi.Height >= 480 {
		return "480p"
	}
	return "SD"
}

// IsHD returns true if the video is considered HD quality
func (vi *VideoInfo) IsHD() bool {
	return vi.Height >= 720
}

// GetAspectRatio calculates the aspect ratio as a float
func (vi *VideoInfo) GetAspectRatio() float64 {
	if vi.Width > 0 && vi.Height > 0 {
		return float64(vi.Width) / float64(vi.Height)
	}
	return 0
}

// GetSubtitleTracks returns information about subtitle tracks in the video file
func GetSubtitleTracks(videoPath string) ([]map[string]string, error) {
	cmd := exec.CommandContext(context.Background(), ffprobePath,
		"-v", "quiet",
		"-select_streams", "s",
		"-print_format", "json",
		"-show_streams",
		videoPath)

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("ffprobe subtitle analysis failed: %w", err)
	}

	var result ffprobeResult
	if err := json.Unmarshal(output, &result); err != nil {
		return nil, fmt.Errorf("failed to parse ffprobe output: %w", err)
	}

	var tracks []map[string]string
	for _, stream := range result.Streams {
		if stream.CodecType == "subtitle" {
			track := map[string]string{
				"index": fmt.Sprintf("%d", stream.Index),
				"codec": stream.CodecName,
			}
			tracks = append(tracks, track)
		}
	}

	return tracks, nil
}

// SubtitleStream describes one embedded subtitle track of a media file.
type SubtitleStream struct {
	// Ordinal is the 0-based index among subtitle streams, suitable for
	// ffmpeg's "0:s:N" stream selector.
	Ordinal int
	// Codec is the ffmpeg codec name, e.g. "subrip", "ass", "hdmv_pgs_subtitle".
	Codec string
	// Language is the ISO language tag on the stream (may be empty).
	Language string
	// Title is the stream title tag (may be empty).
	Title string
}

// ImageBased reports whether the subtitle codec is a bitmap/image format
// (PGS/VobSub/DVD) that cannot be losslessly converted to text.
func (s SubtitleStream) ImageBased() bool {
	switch s.Codec {
	case "hdmv_pgs_subtitle", "pgssub", "dvd_subtitle", "dvdsub", "xsub":
		return true
	default:
		return false
	}
}

// SubtitleStreams lists the embedded subtitle tracks of a media file, including
// their language tag and codec, using ffprobe.
func SubtitleStreams(mediaPath string) ([]SubtitleStream, error) {
	cmd := exec.CommandContext(context.Background(), ffprobePath,
		"-v", "quiet",
		"-select_streams", "s",
		"-print_format", "json",
		"-show_streams",
		mediaPath)
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("ffprobe subtitle streams: %w", err)
	}
	var res struct {
		Streams []struct {
			CodecName string `json:"codec_name"`
			Tags      struct {
				Language string `json:"language"`
				Title    string `json:"title"`
			} `json:"tags"`
		} `json:"streams"`
	}
	if err := json.Unmarshal(out, &res); err != nil {
		return nil, fmt.Errorf("parse ffprobe subtitle streams: %w", err)
	}
	streams := make([]SubtitleStream, 0, len(res.Streams))
	for i, s := range res.Streams {
		streams = append(streams, SubtitleStream{
			Ordinal:  i,
			Codec:    s.CodecName,
			Language: s.Tags.Language,
			Title:    s.Tags.Title,
		})
	}
	return streams, nil
}

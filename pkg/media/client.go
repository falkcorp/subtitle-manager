// file: pkg/media/client.go
// version: 1.0.0
// guid: 789e4567-e89b-12d3-a456-426614174111

// Package media provides client implementations for media processing services
// using the gcommon media protobuf definitions.
package media

import (
	"context"
	"fmt"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	mediapb "github.com/falkcorp/gcommon/v2/pkg/mediapb/v2"
)

// Client wraps gRPC clients for media services
type Client struct {
	mediaService      mediapb.MediaServiceClient
	subtitleService   mediapb.SubtitleServiceClient
	processingService mediapb.MediaProcessingServiceClient
	audioService      mediapb.AudioServiceClient
	conn              *grpc.ClientConn
}

// NewClient creates a new media services client
func NewClient(serverAddr string) (*Client, error) {
	conn, err := grpc.Dial(serverAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to media server: %w", err)
	}

	return &Client{
		mediaService:      mediapb.NewMediaServiceClient(conn),
		subtitleService:   mediapb.NewSubtitleServiceClient(conn),
		processingService: mediapb.NewMediaProcessingServiceClient(conn),
		audioService:      mediapb.NewAudioServiceClient(conn),
		conn:              conn,
	}, nil
}

// Close closes the connection to the media server
func (c *Client) Close() error {
	return c.conn.Close()
}

// CreateMediaFile creates a new media file entry
func (c *Client) CreateMediaFile(ctx context.Context, req *mediapb.CreateMediaFileRequest) (*mediapb.CreateMediaFileResponse, error) {
	return c.mediaService.CreateMediaFile(ctx, req)
}

// GetMediaFile retrieves a media file by ID
func (c *Client) GetMediaFile(ctx context.Context, req *mediapb.GetMediaFileRequest) (*mediapb.GetMediaFileResponse, error) {
	return c.mediaService.GetMediaFile(ctx, req)
}

// ListMediaFiles lists media files with optional filtering
func (c *Client) ListMediaFiles(ctx context.Context, req *mediapb.ListMediaFilesRequest) (*mediapb.ListMediaFilesResponse, error) {
	return c.mediaService.ListMediaFiles(ctx, req)
}

// SearchMedia searches for media files
func (c *Client) SearchMedia(ctx context.Context, req *mediapb.SearchMediaRequest) (*mediapb.SearchMediaResponse, error) {
	return c.mediaService.SearchMedia(ctx, req)
}

// ExtractSubtitles extracts subtitles from media files
func (c *Client) ExtractSubtitles(ctx context.Context, req *mediapb.ExtractSubtitlesRequest) (*mediapb.ExtractSubtitlesResponse, error) {
	return c.processingService.ExtractSubtitles(ctx, req)
}

// ConvertSubtitleFormat converts subtitles between formats
func (c *Client) ConvertSubtitleFormat(ctx context.Context, req *mediapb.ConvertSubtitleFormatRequest) (*mediapb.ConvertSubtitleFormatResponse, error) {
	return c.subtitleService.ConvertSubtitleFormat(ctx, req)
}

// MergeSubtitles merges multiple subtitle tracks
func (c *Client) MergeSubtitles(ctx context.Context, req *mediapb.MergeSubtitlesRequest) (*mediapb.MergeSubtitlesResponse, error) {
	return c.subtitleService.MergeSubtitles(ctx, req)
}

// SyncSubtitles synchronizes subtitle timing
func (c *Client) SyncSubtitles(ctx context.Context, req *mediapb.SyncSubtitlesRequest) (*mediapb.SyncSubtitlesResponse, error) {
	return c.subtitleService.SyncSubtitles(ctx, req)
}

// ValidateSubtitles validates subtitle files
func (c *Client) ValidateSubtitles(ctx context.Context, req *mediapb.ValidateSubtitlesRequest) (*mediapb.ValidateSubtitlesResponse, error) {
	return c.subtitleService.ValidateSubtitles(ctx, req)
}

// AdjustSubtitleTiming adjusts subtitle timing
func (c *Client) AdjustSubtitleTiming(ctx context.Context, req *mediapb.AdjustSubtitleTimingRequest) (*mediapb.AdjustSubtitleTimingResponse, error) {
	return c.subtitleService.AdjustSubtitleTiming(ctx, req)
}

// TranscodeMedia transcodes media files
func (c *Client) TranscodeMedia(ctx context.Context, req *mediapb.TranscodeMediaRequest) (*mediapb.TranscodeMediaResponse, error) {
	return c.processingService.TranscodeMedia(ctx, req)
}

// AnalyzeMedia analyzes media file properties
func (c *Client) AnalyzeMedia(ctx context.Context, req *mediapb.AnalyzeMediaRequest) (*mediapb.AnalyzeMediaResponse, error) {
	return c.processingService.AnalyzeMedia(ctx, req)
}

// GetProcessingStatus gets the status of media processing operations
func (c *Client) GetProcessingStatus(ctx context.Context, req *mediapb.GetProcessingStatusRequest) (*mediapb.GetProcessingStatusResponse, error) {
	return c.processingService.GetProcessingStatus(ctx, req)
}

// ExtractAudio extracts audio from media files
func (c *Client) ExtractAudio(ctx context.Context, req *mediapb.ExtractAudioRequest) (*mediapb.ExtractAudioResponse, error) {
	return c.processingService.ExtractAudio(ctx, req)
}

// NormalizeAudio normalizes audio levels
func (c *Client) NormalizeAudio(ctx context.Context, req *mediapb.NormalizeAudioRequest) (*mediapb.NormalizeAudioResponse, error) {
	return c.audioService.NormalizeAudio(ctx, req)
}

// AnalyzeAudioQuality analyzes audio quality metrics
func (c *Client) AnalyzeAudioQuality(ctx context.Context, req *mediapb.AnalyzeAudioQualityRequest) (*mediapb.AnalyzeAudioQualityResponse, error) {
	return c.audioService.AnalyzeAudioQuality(ctx, req)
}

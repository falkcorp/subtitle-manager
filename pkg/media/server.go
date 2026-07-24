// file: pkg/media/server.go
// version: 1.2.1
// guid: 9a8b7c6d-5e4f-3a2b-1c0d-9e8f7a6b5c4d

package media

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	mediagrpc "buf.build/gen/go/falkcorp/gcommon/grpc/go/mediapb/v2/mediav2grpc"
	media "buf.build/gen/go/falkcorp/gcommon/protocolbuffers/go/mediapb/v2"
	"github.com/asticode/go-astisub"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/jdfalk/subtitle-manager/pkg/audio"
	"github.com/jdfalk/subtitle-manager/pkg/subtitles"
	"github.com/jdfalk/subtitle-manager/pkg/video"
)

// MediaServiceServer implements the media service gRPC server
type MediaServiceServer struct {
	mediagrpc.UnimplementedMediaServiceServer
	fileStorage *FileStorage
}

// SubtitleServiceServer implements the subtitle service gRPC server
type SubtitleServiceServer struct {
	mediagrpc.UnimplementedSubtitleServiceServer
	fileStorage *FileStorage
}

// NewMediaServiceServer creates a new media service server
func NewMediaServiceServer() *MediaServiceServer {
	return &MediaServiceServer{
		fileStorage: NewFileStorage(),
	}
}

// NewSubtitleServiceServer creates a new subtitle service server
func NewSubtitleServiceServer() *SubtitleServiceServer {
	return &SubtitleServiceServer{
		fileStorage: NewFileStorage(),
	}
}

// ExtractSubtitles extracts subtitles from a media file
func (s *SubtitleServiceServer) ExtractSubtitles(ctx context.Context, req *media.ExtractSubtitlesRequest) (*media.ExtractSubtitlesResponse, error) {
	mediaFileId := req.GetMediaFileId()
	trackIndices := req.GetTrackIndices()
	options := req.GetOptions()

	// Get the file path from the file ID
	mediaPath, err := s.fileStorage.GetFilePath(mediaFileId)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "media file not found: %v", err)
	}

	response := &media.ExtractSubtitlesResponse{}
	var extractedSubtitles []*media.ExtractedSubtitle

	// If no specific tracks specified, extract all subtitle tracks
	if len(trackIndices) == 0 {
		// Get available subtitle tracks
		tracks, err := video.GetSubtitleTracks(mediaPath)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to analyze subtitle tracks: %v", err)
		}

		// Extract each track
		for _, track := range tracks {
			trackIndex, _ := strconv.Atoi(track["index"])
			extracted, err := s.extractSingleTrack(mediaPath, trackIndex)
			if err != nil {
				// Log error but continue with other tracks
				continue
			}
			extractedSubtitles = append(extractedSubtitles, extracted)
		}
	} else {
		// Extract specified tracks
		for _, trackIndex := range trackIndices {
			extracted, err := s.extractSingleTrack(mediaPath, int(trackIndex))
			if err != nil {
				return nil, status.Errorf(codes.Internal, "failed to extract track %d: %v", trackIndex, err)
			}
			extractedSubtitles = append(extractedSubtitles, extracted)
		}
	}

	if len(extractedSubtitles) == 0 {
		return nil, status.Errorf(codes.NotFound, "no subtitle tracks found or extracted")
	}

	// Handle extraction options if provided
	_ = options // TODO: Implement format preferences, language filtering, etc.

	response.SetStatus("success")
	response.SetExtractedSubtitles(extractedSubtitles)

	return response, nil
}

// extractSingleTrack extracts a single subtitle track using the existing subtitles package
func (s *SubtitleServiceServer) extractSingleTrack(mediaPath string, trackIndex int) (*media.ExtractedSubtitle, error) {
	// Use the existing subtitle extraction functionality
	items, err := subtitles.ExtractTrack(mediaPath, trackIndex)
	if err != nil {
		return nil, fmt.Errorf("ffmpeg extraction failed: %w", err)
	}

	// Create temporary file for the extracted subtitles
	fileID, outputPath, err := s.fileStorage.CreateTempFile("extracted", ".srt")
	if err != nil {
		return nil, fmt.Errorf("failed to create output file: %w", err)
	}

	// Create output file
	outputFile, err := os.Create(outputPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create output file: %w", err)
	}
	defer outputFile.Close()

	// Create astisub object and write to file
	sub := &astisub.Subtitles{Items: items}
	if err := sub.WriteToSRT(outputFile); err != nil {
		return nil, fmt.Errorf("failed to write subtitle file: %w", err)
	}

	// Create response object
	extracted := &media.ExtractedSubtitle{}
	extracted.SetFileId(fileID)
	extracted.SetTrackIndex(int32(trackIndex))
	extracted.SetFormat("srt")
	extracted.SetLanguage("unknown") // TODO: Detect language from metadata

	return extracted, nil
}

// ConvertSubtitleFormat converts subtitle format
func (s *SubtitleServiceServer) ConvertSubtitleFormat(ctx context.Context, req *media.ConvertSubtitleFormatRequest) (*media.ConvertSubtitleFormatResponse, error) {
	subtitleFileId := req.GetSubtitleFileId()
	targetFormat := req.GetTargetFormat()
	_ = req.GetPreserveStyling() // Currently unused

	// For demonstration purposes, assume we have the file path from the ID
	inputPath := fmt.Sprintf("/tmp/subtitles/%s", subtitleFileId)
	outputPath := fmt.Sprintf("/tmp/subtitles/%s_converted.%s", subtitleFileId, targetFormat)

	// Read the subtitle file
	subtitles, err := astisub.OpenFile(inputPath)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "failed to open subtitle file %s: %v", inputPath, err)
	}

	// Create output file
	outputFile, err := os.Create(outputPath)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create output file: %v", err)
	}
	defer outputFile.Close()

	// Convert to target format and write
	switch strings.ToLower(targetFormat) {
	case "srt":
		err = subtitles.WriteToSRT(outputFile)
	case "ass", "ssa":
		err = subtitles.WriteToSSA(outputFile)
	case "vtt":
		err = subtitles.WriteToWebVTT(outputFile)
	case "ttml":
		err = subtitles.WriteToTTML(outputFile)
	default:
		return nil, status.Errorf(codes.InvalidArgument, "unsupported target format: %s", targetFormat)
	}

	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to write converted subtitles: %v", err)
	}

	response := &media.ConvertSubtitleFormatResponse{}
	response.SetSuccess(true)
	response.SetConvertedSubtitleFileId(fmt.Sprintf("%s_converted", subtitleFileId))
	response.SetOutputFormat(targetFormat)

	return response, nil
}

// MergeSubtitles merges multiple subtitle files
func (s *SubtitleServiceServer) MergeSubtitles(ctx context.Context, req *media.MergeSubtitlesRequest) (*media.MergeSubtitlesResponse, error) {
	subtitleFileIds := req.GetSubtitleFileIds()
	mergeOptions := req.GetMergeOptions()

	if len(subtitleFileIds) < 2 {
		response := &media.MergeSubtitlesResponse{}
		response.SetSuccess(false)
		response.SetErrorMessage("At least two subtitle files are required for merging")
		return response, nil
	}

	// Load all subtitle files
	var subtitleFiles []*astisub.Subtitles
	for _, fileID := range subtitleFileIds {
		filePath, err := s.fileStorage.GetFilePath(fileID)
		if err != nil {
			response := &media.MergeSubtitlesResponse{}
			response.SetSuccess(false)
			response.SetErrorMessage(fmt.Sprintf("Failed to find subtitle file: %v", err))
			return response, nil
		}

		sub, err := astisub.OpenFile(filePath)
		if err != nil {
			response := &media.MergeSubtitlesResponse{}
			response.SetSuccess(false)
			response.SetErrorMessage(fmt.Sprintf("Failed to load subtitle file %s: %v", filePath, err))
			return response, nil
		}
		subtitleFiles = append(subtitleFiles, sub)
	}

	// Start with the first subtitle file as base
	merged := subtitleFiles[0]

	// Merge additional subtitle files
	for i := 1; i < len(subtitleFiles); i++ {
		sub := subtitleFiles[i]

		// Apply time offset if specified in merge options
		timeOffset := time.Duration(0)
		if mergeOptions != nil {
			// TODO: Extract time offset from merge options when available
		}

		// Add items from this subtitle file with time offset
		for _, item := range sub.Items {
			newItem := *item // Copy the item
			newItem.StartAt += timeOffset
			newItem.EndAt += timeOffset
			merged.Items = append(merged.Items, &newItem)
		}
	}

	// Sort items by start time to maintain proper order
	merged.Order()

	// Create output file
	outputFileID, outputPath, err := s.fileStorage.CreateTempFile("merged", ".srt")
	if err != nil {
		response := &media.MergeSubtitlesResponse{}
		response.SetSuccess(false)
		response.SetErrorMessage(fmt.Sprintf("Failed to create output file: %v", err))
		return response, nil
	}

	// Write merged subtitles
	outputFile, err := os.Create(outputPath)
	if err != nil {
		response := &media.MergeSubtitlesResponse{}
		response.SetSuccess(false)
		response.SetErrorMessage(fmt.Sprintf("Failed to create output file: %v", err))
		return response, nil
	}
	defer outputFile.Close()

	if err := merged.WriteToSRT(outputFile); err != nil {
		response := &media.MergeSubtitlesResponse{}
		response.SetSuccess(false)
		response.SetErrorMessage(fmt.Sprintf("Failed to write merged subtitle file: %v", err))
		return response, nil
	}

	response := &media.MergeSubtitlesResponse{}
	response.SetSuccess(true)
	response.SetMergedSubtitleFileId(outputFileID)
	return response, nil
}

// SyncSubtitles synchronizes subtitle timing
func (s *SubtitleServiceServer) SyncSubtitles(ctx context.Context, req *media.SyncSubtitlesRequest) (*media.SyncSubtitlesResponse, error) {
	mediaFileId := req.GetMediaFileId()
	subtitleFileId := req.GetSubtitleFileId()
	autoDetectTiming := req.GetAutoDetectTiming()
	syncPointsMs := req.GetSyncPointsMs()

	// Get file paths
	mediaPath, err := s.fileStorage.GetFilePath(mediaFileId)
	if err != nil {
		response := &media.SyncSubtitlesResponse{}
		response.SetSuccess(false)
		response.SetErrorMessage(fmt.Sprintf("Media file not found: %v", err))
		return response, nil
	}

	subtitlePath, err := s.fileStorage.GetFilePath(subtitleFileId)
	if err != nil {
		response := &media.SyncSubtitlesResponse{}
		response.SetSuccess(false)
		response.SetErrorMessage(fmt.Sprintf("Subtitle file not found: %v", err))
		return response, nil
	}

	// Load subtitle file
	sub, err := astisub.OpenFile(subtitlePath)
	if err != nil {
		response := &media.SyncSubtitlesResponse{}
		response.SetSuccess(false)
		response.SetErrorMessage(fmt.Sprintf("Failed to load subtitle file: %v", err))
		return response, nil
	}

	if autoDetectTiming {
		// TODO: Implement automatic timing detection using audio analysis
		// This would involve:
		// 1. Extract audio from video using pkg/audio
		// 2. Perform speech detection/analysis
		// 3. Align subtitle timing with detected speech
		// For now, we'll skip this complex feature
	} else if len(syncPointsMs) >= 2 {
		// Apply linear correction using sync points
		// Convert sync points from milliseconds to time.Duration
		if len(syncPointsMs) >= 4 {
			actual1 := time.Duration(syncPointsMs[0]) * time.Millisecond
			desired1 := time.Duration(syncPointsMs[1]) * time.Millisecond
			actual2 := time.Duration(syncPointsMs[2]) * time.Millisecond
			desired2 := time.Duration(syncPointsMs[3]) * time.Millisecond

			sub.ApplyLinearCorrection(actual1, desired1, actual2, desired2)
		} else {
			// Simple offset adjustment
			offset := time.Duration(syncPointsMs[1]-syncPointsMs[0]) * time.Millisecond
			sub.Add(offset)
		}
	}

	// Create output file for synchronized subtitles
	outputFileID, outputPath, err := s.fileStorage.CreateTempFile("synced", ".srt")
	if err != nil {
		response := &media.SyncSubtitlesResponse{}
		response.SetSuccess(false)
		response.SetErrorMessage(fmt.Sprintf("Failed to create output file: %v", err))
		return response, nil
	}

	// Write synchronized subtitles
	outputFile, err := os.Create(outputPath)
	if err != nil {
		response := &media.SyncSubtitlesResponse{}
		response.SetSuccess(false)
		response.SetErrorMessage(fmt.Sprintf("Failed to create output file: %v", err))
		return response, nil
	}
	defer outputFile.Close()

	if err := sub.WriteToSRT(outputFile); err != nil {
		response := &media.SyncSubtitlesResponse{}
		response.SetSuccess(false)
		response.SetErrorMessage(fmt.Sprintf("Failed to write synchronized subtitle file: %v", err))
		return response, nil
	}

	response := &media.SyncSubtitlesResponse{}
	response.SetSuccess(true)
	response.SetSynchronizedSubtitleFileId(outputFileID)

	// Use provided parameters to avoid unused variable errors
	_ = mediaPath

	return response, nil
}

// ValidateSubtitles validates subtitle file format and content
func (s *SubtitleServiceServer) ValidateSubtitles(ctx context.Context, req *media.ValidateSubtitlesRequest) (*media.ValidateSubtitlesResponse, error) {
	subtitleFileId := req.GetSubtitleFileId()
	checkFormatting := req.GetCheckFormatting()
	checkTiming := req.GetCheckTiming()
	expectedFormat := req.GetExpectedFormat()

	// Get subtitle file path
	subtitlePath, err := s.fileStorage.GetFilePath(subtitleFileId)
	if err != nil {
		response := &media.ValidateSubtitlesResponse{}
		response.SetIsValid(false)
		response.SetValidationErrors([]string{fmt.Sprintf("Subtitle file not found: %v", err)})
		return response, nil
	}

	// Load subtitle file
	sub, err := astisub.OpenFile(subtitlePath)
	if err != nil {
		response := &media.ValidateSubtitlesResponse{}
		response.SetIsValid(false)
		response.SetValidationErrors([]string{fmt.Sprintf("Failed to parse subtitle file: %v", err)})
		return response, nil
	}

	var validationErrors []string
	var warnings []string

	// Basic content validation
	if len(sub.Items) == 0 {
		validationErrors = append(validationErrors, "Subtitle file is empty")
	}

	// Format validation
	if checkFormatting {
		for i, item := range sub.Items {
			// Check for empty text
			if len(strings.TrimSpace(item.String())) == 0 {
				warnings = append(warnings, fmt.Sprintf("Item %d: empty text", i+1))
			}

			// Check for extremely long lines
			lines := strings.Split(item.String(), "\n")
			for lineNum, line := range lines {
				if len(line) > 120 {
					warnings = append(warnings, fmt.Sprintf("Item %d, line %d: very long line (%d characters)", i+1, lineNum+1, len(line)))
				}
			}
		}
	}

	// Timing validation
	if checkTiming {
		for i, item := range sub.Items {
			// Check for invalid timing
			if item.StartAt >= item.EndAt {
				validationErrors = append(validationErrors, fmt.Sprintf("Item %d: start time >= end time", i+1))
			}

			// Check for extremely short duration
			duration := item.EndAt - item.StartAt
			if duration < 500*time.Millisecond {
				warnings = append(warnings, fmt.Sprintf("Item %d: very short duration (%.1fs)", i+1, duration.Seconds()))
			}

			// Check for overlapping with next item
			if i < len(sub.Items)-1 {
				nextItem := sub.Items[i+1]
				if item.EndAt > nextItem.StartAt {
					warnings = append(warnings, fmt.Sprintf("Item %d overlaps with item %d", i+1, i+2))
				}
			}
		}
	}

	// Format-specific validation
	if expectedFormat != "" {
		// Check if file extension matches expected format
		actualFormat := strings.ToLower(filepath.Ext(subtitlePath))
		if actualFormat != "" {
			actualFormat = actualFormat[1:] // Remove leading dot
		}
		if expectedFormat != actualFormat {
			warnings = append(warnings, fmt.Sprintf("Expected format %s but got %s", expectedFormat, actualFormat))
		}
	}

	response := &media.ValidateSubtitlesResponse{}
	response.SetIsValid(len(validationErrors) == 0)

	if len(validationErrors) > 0 {
		response.SetValidationErrors(validationErrors)
	}

	if len(warnings) > 0 {
		response.SetValidationWarnings(warnings)
	}

	// Detect format from file extension
	detectedFormat := strings.ToLower(filepath.Ext(subtitlePath))
	if detectedFormat != "" {
		detectedFormat = detectedFormat[1:] // Remove leading dot
		response.SetDetectedFormat(detectedFormat)
	}

	return response, nil
}

// AdjustSubtitleTiming adjusts subtitle timing
func (s *SubtitleServiceServer) AdjustSubtitleTiming(ctx context.Context, req *media.AdjustSubtitleTimingRequest) (*media.AdjustSubtitleTimingResponse, error) {
	// Check what methods are available on the request
	// For now, return a basic response
	response := &media.AdjustSubtitleTimingResponse{}
	response.SetSuccess(true)
	return response, nil
}

// CreateMediaFile creates a new media file record
func (s *MediaServiceServer) CreateMediaFile(ctx context.Context, req *media.CreateMediaFileRequest) (*media.CreateMediaFileResponse, error) {
	// Check what methods are available on the request
	// For now, return a basic response
	response := &media.CreateMediaFileResponse{}

	// Create a media file
	mediaFile := &media.MediaFile{}
	mediaFile.SetId("media-file-123")
	mediaFile.SetFilename("example.mp4")

	response.SetMediaFile(mediaFile)
	return response, nil
}

// GetMediaFile retrieves a media file by ID
func (s *MediaServiceServer) GetMediaFile(ctx context.Context, req *media.GetMediaFileRequest) (*media.GetMediaFileResponse, error) {
	// Check what methods are available on the request
	// For now, return a basic response
	response := &media.GetMediaFileResponse{}

	// Create a media file
	mediaFile := &media.MediaFile{}
	mediaFile.SetId("media-file-123")
	mediaFile.SetFilename("example.mp4")

	response.SetMediaFile(mediaFile)
	return response, nil
}

// ListMediaFiles lists media files
func (s *MediaServiceServer) ListMediaFiles(ctx context.Context, req *media.ListMediaFilesRequest) (*media.ListMediaFilesResponse, error) {
	// Check what methods are available on the request
	// For now, return a basic response
	response := &media.ListMediaFilesResponse{}

	// Create a media file
	mediaFile := &media.MediaFile{}
	mediaFile.SetId("media-file-123")
	mediaFile.SetFilename("example.mp4")

	response.SetMediaFiles([]*media.MediaFile{mediaFile})
	return response, nil
}

// UpdateMediaFile updates a media file
func (s *MediaServiceServer) UpdateMediaFile(ctx context.Context, req *media.UpdateMediaFileRequest) (*media.UpdateMediaFileResponse, error) {
	// Check what methods are available on the request
	// For now, return a basic response
	response := &media.UpdateMediaFileResponse{}
	return response, nil
}

// DeleteMediaFile deletes a media file
func (s *MediaServiceServer) DeleteMediaFile(ctx context.Context, req *media.DeleteMediaFileRequest) (*media.DeleteMediaFileResponse, error) {
	// Check what methods are available on the request
	// For now, return a basic response
	response := &media.DeleteMediaFileResponse{}
	response.SetSuccess(true)
	return response, nil
}

// SearchMedia searches for media files
func (s *MediaServiceServer) SearchMedia(ctx context.Context, req *media.SearchMediaRequest) (*media.SearchMediaResponse, error) {
	// Check what methods are available on the request
	// For now, return a basic response
	response := &media.SearchMediaResponse{}

	// Create a media file
	mediaFile := &media.MediaFile{}
	mediaFile.SetId("media-file-123")
	mediaFile.SetFilename("example.mp4")

	response.SetMediaFiles([]*media.MediaFile{mediaFile})
	return response, nil
}

// UploadMedia handles media file uploads
func (s *MediaServiceServer) UploadMedia(ctx context.Context, req *media.UploadMediaRequest) (*media.UploadMediaResponse, error) {
	// Check what methods are available on the request
	// For now, return a basic response
	response := &media.UploadMediaResponse{}

	// Create a media file
	mediaFile := &media.MediaFile{}
	mediaFile.SetId("media-file-123")
	mediaFile.SetFilename("uploaded.mp4")

	response.SetMediaFile(mediaFile)
	return response, nil
}

// NewServer creates a new server instance
type Server struct {
	mediagrpc.UnimplementedMediaServiceServer
	mediagrpc.UnimplementedSubtitleServiceServer
	mediagrpc.UnimplementedMediaProcessingServiceServer
	mediagrpc.UnimplementedAudioServiceServer

	// Configuration
	mediaRoot    string
	subtitleRoot string
	tempDir      string
}

// NewServer creates a new media server instance
func NewServer(mediaRoot, subtitleRoot, tempDir string) *Server {
	return &Server{
		mediaRoot:    mediaRoot,
		subtitleRoot: subtitleRoot,
		tempDir:      tempDir,
	}
}

// CreateMediaFile creates a new media file entry
func (s *Server) CreateMediaFile(ctx context.Context, req *media.CreateMediaFileRequest) (*media.CreateMediaFileResponse, error) {
	inputFile := req.GetMediaFile()
	if inputFile == nil {
		return &media.CreateMediaFileResponse{}, fmt.Errorf("media file is required")
	}

	// Generate a unique ID for the media file if not provided
	fileID := inputFile.GetId()
	if fileID == "" {
		fileID = uuid.New().String()
	}

	// Create media file metadata with existing getters
	// Note: We can't directly set fields due to protobuf opaque generation
	// This would need proper protobuf message construction

	return &media.CreateMediaFileResponse{}, nil
}

// GetMediaFile retrieves a media file by ID
func (s *Server) GetMediaFile(ctx context.Context, req *media.GetMediaFileRequest) (*media.GetMediaFileResponse, error) {
	// In a real implementation, this would query a database
	return &media.GetMediaFileResponse{}, nil
}

// ListMediaFiles lists media files with optional filtering
func (s *Server) ListMediaFiles(ctx context.Context, req *media.ListMediaFilesRequest) (*media.ListMediaFilesResponse, error) {
	// In a real implementation, this would query a database with pagination
	return &media.ListMediaFilesResponse{}, nil
}

// SearchMedia searches for media files
func (s *Server) SearchMedia(ctx context.Context, req *media.SearchMediaRequest) (*media.SearchMediaResponse, error) {
	// In a real implementation, this would perform full-text search
	return &media.SearchMediaResponse{}, nil
}

// ExtractSubtitles extracts subtitles from media files
func (s *Server) ExtractSubtitles(ctx context.Context, req *media.ExtractSubtitlesRequest) (*media.ExtractSubtitlesResponse, error) {
	mediaFileId := req.GetMediaFileId()
	trackIndices := req.GetTrackIndices()
	options := req.GetOptions()

	// In a real implementation, this would use ffmpeg or similar to extract subtitles
	// For now, we'll simulate success
	_ = mediaFileId
	_ = trackIndices
	_ = options

	return &media.ExtractSubtitlesResponse{}, nil
}

// MergeSubtitles merges multiple subtitle tracks
func (s *Server) MergeSubtitles(ctx context.Context, req *media.MergeSubtitlesRequest) (*media.MergeSubtitlesResponse, error) {
	subtitleFileIds := req.GetSubtitleFileIds()
	outputFileId := req.GetOutputFileId()

	if len(subtitleFileIds) < 2 {
		response := &media.MergeSubtitlesResponse{}
		response.SetSuccess(false)
		response.SetErrorMessage("At least two subtitle files are required for merging")
		return response, nil
	}

	// TODO: Implement actual file merging with ID-based file system
	// For now, return success with the output file ID
	response := &media.MergeSubtitlesResponse{}
	response.SetSuccess(true)
	response.SetMergedSubtitleFileId(outputFileId)
	return response, nil
}

// SyncSubtitles synchronizes subtitle timing
func (s *Server) SyncSubtitles(ctx context.Context, req *media.SyncSubtitlesRequest) (*media.SyncSubtitlesResponse, error) {
	mediaFileId := req.GetMediaFileId()
	subtitleFileId := req.GetSubtitleFileId()
	autoDetectTiming := req.GetAutoDetectTiming()
	syncPointsMs := req.GetSyncPointsMs()

	// TODO: Implement actual subtitle synchronization with ID-based file system
	// For now, return success
	response := &media.SyncSubtitlesResponse{}
	response.SetSuccess(true)

	// Use provided parameters to avoid unused variable errors
	_ = mediaFileId
	_ = subtitleFileId
	_ = autoDetectTiming
	_ = syncPointsMs

	return response, nil
}

// ValidateSubtitles validates subtitle files
func (s *Server) ValidateSubtitles(ctx context.Context, req *media.ValidateSubtitlesRequest) (*media.ValidateSubtitlesResponse, error) {
	subtitleFileId := req.GetSubtitleFileId()
	checkFormatting := req.GetCheckFormatting()
	checkTiming := req.GetCheckTiming()
	expectedFormat := req.GetExpectedFormat()

	// TODO: Implement actual subtitle validation with ID-based file system
	// For now, return success
	response := &media.ValidateSubtitlesResponse{}
	response.SetIsValid(true)

	// Use provided parameters to avoid unused variable errors
	_ = subtitleFileId
	_ = checkFormatting
	_ = checkTiming
	_ = expectedFormat

	return response, nil
}

// AdjustSubtitleTiming adjusts subtitle timing
func (s *Server) AdjustSubtitleTiming(ctx context.Context, req *media.AdjustSubtitleTimingRequest) (*media.AdjustSubtitleTimingResponse, error) {
	return &media.AdjustSubtitleTimingResponse{}, nil
}

func (s *Server) TranscodeMedia(ctx context.Context, req *media.TranscodeMediaRequest) (*media.TranscodeMediaResponse, error) {
	mediaFileId := req.GetMediaFileId()
	outputFormat := req.GetOutputFormat()
	outputCodec := req.GetOutputCodec()

	// Get the file path from the file ID
	mediaPath := fmt.Sprintf("/tmp/media/%s", mediaFileId) // Placeholder path mapping

	// TODO: Implement real media transcoding using ffmpeg
	// This would involve:
	// 1. Validating input format and codec compatibility
	// 2. Setting up ffmpeg transcoding parameters based on output format/codec
	// 3. Running ffmpeg with appropriate options for quality, bitrate, etc.
	// 4. Monitoring progress and providing status updates
	//
	// Example ffmpeg command for transcoding:
	// ffmpeg -i input.mp4 -c:v libx264 -c:a aac -b:v 2M -b:a 128k output.mp4

	// Generate job ID for tracking
	jobId := uuid.New().String()

	// Generate output file ID
	outputFileId := fmt.Sprintf("%s_transcoded_%s", mediaFileId, outputFormat)

	// Create response indicating transcoding is started
	response := &media.TranscodeMediaResponse{}
	response.SetStatus("started")
	response.SetJobId(jobId)
	response.SetOutputFileId(outputFileId)
	response.SetProgressPercent(0)

	// Note: In production, this would start a background transcoding job
	// and return immediately, with progress checkable via GetProcessingStatus
	_ = mediaPath
	_ = outputCodec

	return response, nil
}

func (s *Server) AnalyzeMedia(ctx context.Context, req *media.AnalyzeMediaRequest) (*media.AnalyzeMediaResponse, error) {
	mediaFileId := req.GetMediaFileId()

	// Get the file path from the file ID
	// Note: Server struct needs fileStorage too, but for now we'll create a basic implementation
	mediaPath := fmt.Sprintf("/tmp/media/%s", mediaFileId) // Placeholder path mapping

	// Analyze video using our video package
	videoInfo, err := video.AnalyzeVideo(mediaPath)
	if err != nil {
		response := &media.AnalyzeMediaResponse{}
		response.SetStatus("error")
		response.SetErrorMessage(fmt.Sprintf("Failed to analyze video: %v", err))
		return response, nil
	}

	// Get audio tracks
	audioTracks, err := audio.GetAudioTracks(mediaPath)
	if err != nil {
		// Log error but continue - audio analysis is optional
		audioTracks = []map[string]string{}
	}

	// Get subtitle tracks - for now, just create empty slice since we need to implement this
	subtitleTracks := []map[string]string{}

	// Create response
	response := &media.AnalyzeMediaResponse{}
	response.SetStatus("completed")

	// Create the media analysis
	analysis := &media.MediaAnalysis{}

	// Create technical metadata
	technical := &media.TechnicalMetadata{}
	technical.SetContainerFormat(videoInfo.Format)
	technical.SetFileSize(videoInfo.FileSize)

	// Set duration using duration protobuf
	duration := &durationpb.Duration{
		Seconds: int64(videoInfo.Duration.Seconds()),
		Nanos:   int32(videoInfo.Duration.Nanoseconds() % 1e9),
	}
	technical.SetDuration(duration)

	// Set overall bitrate
	technical.SetBitrate(videoInfo.Bitrate)

	// Create video stream info
	videoStream := &media.VideoStreamInfo{}
	videoStream.SetWidth(int32(videoInfo.Width))
	videoStream.SetHeight(int32(videoInfo.Height))
	videoStream.SetCodec(videoInfo.Codec)
	videoStream.SetFrameRate(videoInfo.FrameRate)
	videoStream.SetBitrate(videoInfo.Bitrate)
	// Note: PixelFormat not available in current VideoInfo struct
	technical.SetVideo(videoStream)

	// Add audio streams if available
	if len(audioTracks) > 0 {
		audioStreams := make([]*media.AudioStreamInfo, len(audioTracks))
		for i, track := range audioTracks {
			audioStream := &media.AudioStreamInfo{}
			if codec, ok := track["codec"]; ok {
				audioStream.SetCodec(codec)
			}
			if bitrate, ok := track["bitrate"]; ok {
				if br, err := strconv.ParseInt(bitrate, 10, 64); err == nil {
					audioStream.SetBitrate(br)
				}
			}
			if sampleRate, ok := track["sample_rate"]; ok {
				if sr, err := strconv.ParseInt(sampleRate, 10, 32); err == nil {
					audioStream.SetSampleRate(int32(sr))
				}
			}
			if channels, ok := track["channels"]; ok {
				if ch, err := strconv.ParseInt(channels, 10, 32); err == nil {
					audioStream.SetChannels(int32(ch))
				}
			}
			if language, ok := track["language"]; ok {
				audioStream.SetLanguage(language)
			}
			audioStreams[i] = audioStream
		}
		technical.SetAudioStreams(audioStreams)
	}

	// Add subtitle streams if available
	if len(subtitleTracks) > 0 {
		subtitleStreams := make([]*media.SubtitleStreamInfo, len(subtitleTracks))
		for i, track := range subtitleTracks {
			subtitleStream := &media.SubtitleStreamInfo{}
			if codec, ok := track["codec"]; ok {
				subtitleStream.SetCodec(codec)
			}
			if language, ok := track["language"]; ok {
				subtitleStream.SetLanguage(language)
			}
			subtitleStreams[i] = subtitleStream
		}
		technical.SetSubtitleStreams(subtitleStreams)
	}

	analysis.SetTechnical(technical)
	response.SetAnalysis(analysis)

	return response, nil
}

func (s *Server) GetProcessingStatus(ctx context.Context, req *media.GetProcessingStatusRequest) (*media.GetProcessingStatusResponse, error) {
	jobId := req.GetJobId()

	// TODO: Implement real job status tracking
	// In a production system, this would:
	// 1. Look up the job in a database or job queue
	// 2. Return the actual status, progress, and metadata
	// 3. Handle job states like: queued, running, completed, failed

	// For now, return a mock response based on job ID pattern
	response := &media.GetProcessingStatusResponse{}
	response.SetJobId(jobId)

	// Mock different job types and statuses based on job ID prefix
	if strings.Contains(jobId, "transcode") {
		response.SetJobType("transcoding")
		response.SetStatus("completed")
		response.SetProgressPercent(100)
		response.SetOutputFileIds([]string{fmt.Sprintf("%s_output", jobId)})
	} else if strings.Contains(jobId, "audio") {
		response.SetJobType("audio_extraction")
		response.SetStatus("completed")
		response.SetProgressPercent(100)
		response.SetOutputFileIds([]string{fmt.Sprintf("%s_audio", jobId)})
	} else {
		response.SetJobType("unknown")
		response.SetStatus("not_found")
		response.SetProgressPercent(0)
		response.SetErrorMessage("Job not found")
	}

	// Set timestamps to current time for demonstration
	now := time.Now()
	createdAt := &timestamppb.Timestamp{
		Seconds: now.Add(-5 * time.Minute).Unix(),
		Nanos:   0,
	}
	updatedAt := &timestamppb.Timestamp{
		Seconds: now.Unix(),
		Nanos:   0,
	}
	completedAt := &timestamppb.Timestamp{
		Seconds: now.Unix(),
		Nanos:   0,
	}

	response.SetCreatedAt(createdAt)
	response.SetUpdatedAt(updatedAt)
	if response.GetStatus() == "completed" {
		response.SetCompletedAt(completedAt)
	}

	return response, nil
}

func (s *Server) ExtractAudio(ctx context.Context, req *media.ExtractAudioRequest) (*media.ExtractAudioResponse, error) {
	mediaFileId := req.GetMediaFileId()
	trackIndices := req.GetTrackIndices()

	// Get the file path from the file ID
	mediaPath := fmt.Sprintf("/tmp/media/%s", mediaFileId) // Placeholder path mapping

	// If no track indices specified, extract first audio track
	if len(trackIndices) == 0 {
		trackIndices = []int32{0}
	}

	var outputFileIds []string

	// Extract each requested audio track
	for _, trackIndex := range trackIndices {
		// Use our audio package to extract audio track
		outputPath, err := audio.ExtractTrack(mediaPath, int(trackIndex))
		if err != nil {
			response := &media.ExtractAudioResponse{}
			response.SetStatus("error")
			response.SetErrorMessage(fmt.Sprintf("Failed to extract audio track %d: %v", trackIndex, err))
			return response, nil
		}

		// Generate file ID for the extracted audio
		audioFileId := fmt.Sprintf("%s_audio_track_%d", mediaFileId, trackIndex)
		outputFileIds = append(outputFileIds, audioFileId)

		// In a real implementation, we would register the output path with the file storage
		// For now, just note that outputPath contains the temporary file path
		_ = outputPath
	}

	// Create response
	response := &media.ExtractAudioResponse{}
	response.SetStatus("completed")
	response.SetOutputFileIds(outputFileIds)

	return response, nil
}

func (s *Server) NormalizeAudio(ctx context.Context, req *media.NormalizeAudioRequest) (*media.NormalizeAudioResponse, error) {
	audioFileId := req.GetAudioFileId()

	// Get the file path from the file ID
	audioPath := fmt.Sprintf("/tmp/audio/%s", audioFileId) // Placeholder path mapping

	// TODO: Implement real audio normalization using ffmpeg with loudnorm filter
	// This would involve:
	// 1. Analyzing audio to get current LUFS level
	// 2. Applying ffmpeg loudnorm filter to normalize to target level (e.g., -16 LUFS)
	// 3. Measuring the applied gain and final LUFS level
	//
	// Example ffmpeg command for loudnorm:
	// ffmpeg -i input.wav -af loudnorm=I=-16:TP=-1.5:LRA=11:print_format=json output.wav

	// For now, return a placeholder response indicating the feature needs implementation
	response := &media.NormalizeAudioResponse{}

	// Create a normalized file ID
	normalizedFileId := fmt.Sprintf("%s_normalized", audioFileId)
	response.SetNormalizedAudioFileId(normalizedFileId)

	// Placeholder values - in real implementation these would come from ffmpeg analysis
	response.SetOriginalLufs(-23.0)    // Example original loudness
	response.SetNormalizedLufs(-16.0)  // Target loudness (broadcast standard)
	response.SetGainAppliedDb(7.0)     // Calculated gain applied
	response.SetLimitingApplied(false) // Whether limiting was needed

	// Note: In production, we would actually process the audio file here
	_ = audioPath

	return response, nil
}

func (s *Server) AnalyzeAudioQuality(ctx context.Context, req *media.AnalyzeAudioQualityRequest) (*media.AnalyzeAudioQualityResponse, error) {
	return &media.AnalyzeAudioQualityResponse{}, nil
}

// Additional placeholder methods
func (s *Server) MergeAudio(ctx context.Context, req *media.MergeAudioRequest) (*media.MergeAudioResponse, error) {
	return &media.MergeAudioResponse{}, nil
}

func (s *Server) SplitAudio(ctx context.Context, req *media.SplitAudioRequest) (*media.SplitAudioResponse, error) {
	return &media.SplitAudioResponse{}, nil
}

func (s *Server) UpdateMediaFile(ctx context.Context, req *media.UpdateMediaFileRequest) (*media.UpdateMediaFileResponse, error) {
	return &media.UpdateMediaFileResponse{}, nil
}

func (s *Server) DeleteMediaFile(ctx context.Context, req *media.DeleteMediaFileRequest) (*media.DeleteMediaFileResponse, error) {
	return &media.DeleteMediaFileResponse{}, nil
}

func (s *Server) UploadMedia(ctx context.Context, req *media.UploadMediaRequest) (*media.UploadMediaResponse, error) {
	return &media.UploadMediaResponse{}, nil
}

func (s *Server) DetectChapters(ctx context.Context, req *media.DetectChaptersRequest) (*media.DetectChaptersResponse, error) {
	return &media.DetectChaptersResponse{}, nil
}

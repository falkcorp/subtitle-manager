// file: pkg/transcriber/whisper_container.go
// version: 1.0.0
// guid: 123e4567-e89b-12d3-a456-426614174001

package transcriber

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
	"github.com/google/uuid"
	"github.com/spf13/viper"

	"github.com/jdfalk/subtitle-manager/pkg/security"
	"github.com/jdfalk/subtitle-manager/pkg/tasks"
)

// WhisperContainer manages a local Whisper ASR Docker container.
type WhisperContainer struct {
	client *client.Client
	config ContainerConfig
}

// ContainerConfig holds configuration for the Whisper container.
type ContainerConfig struct {
	ContainerName string
	Image         string
	Port          string
	Model         string
	Device        string
	UseGPU        bool
}

// TranscriptionJob represents a transcription task.
type TranscriptionJob struct {
	ID       string `json:"id"`
	FilePath string `json:"file_path"`
	Language string `json:"language"`
	Model    string `json:"model"`
	Status   string `json:"status"`
	Progress int    `json:"progress"`
	Result   string `json:"result"`
	Error    string `json:"error"`
}

var (
	// SupportedModels are the available Whisper model sizes
	SupportedModels = []string{"tiny", "base", "small", "medium", "large"}

	// DefaultConfig provides sensible defaults
	DefaultConfig = ContainerConfig{
		ContainerName: "whisper-asr-service",
		Image:         "onerahmet/openai-whisper-asr-webservice:latest",
		Port:          "9000",
		Model:         "base",
		Device:        "cuda",
		UseGPU:        true,
	}
)

// NewWhisperContainer creates a new WhisperContainer instance.
func NewWhisperContainer() (*WhisperContainer, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, err
	}
	config := loadContainerConfig()
	return &WhisperContainer{client: cli, config: config}, nil
}

// loadContainerConfig loads configuration from Viper with defaults.
func loadContainerConfig() ContainerConfig {
	return ContainerConfig{
		ContainerName: viper.GetString("whisper.container_name"),
		Image:         viper.GetString("whisper.image"),
		Port:          viper.GetString("whisper.port"),
		Model:         viper.GetString("whisper.model"),
		Device:        viper.GetString("whisper.device"),
		UseGPU:        viper.GetBool("whisper.use_gpu"),
	}
}

// SetDefaultConfig sets default configuration values in Viper.
func SetDefaultConfig() {
	viper.SetDefault("whisper.container_name", DefaultConfig.ContainerName)
	viper.SetDefault("whisper.image", DefaultConfig.Image)
	viper.SetDefault("whisper.port", DefaultConfig.Port)
	viper.SetDefault("whisper.model", DefaultConfig.Model)
	viper.SetDefault("whisper.device", DefaultConfig.Device)
	viper.SetDefault("whisper.use_gpu", DefaultConfig.UseGPU)
}

// IsContainerRunning checks if the Whisper container is currently running.
func (w *WhisperContainer) IsContainerRunning(ctx context.Context) (bool, error) {
	containers, err := w.client.ContainerList(ctx, container.ListOptions{})
	if err != nil {
		return false, fmt.Errorf("failed to list containers: %w", err)
	}

	for _, c := range containers {
		for _, name := range c.Names {
			if strings.TrimPrefix(name, "/") == w.config.ContainerName {
				return c.State == "running", nil
			}
		}
	}
	return false, nil
}

// StartContainer starts the Whisper ASR container.
func (w *WhisperContainer) StartContainer(ctx context.Context) error {
	// Check if container already exists
	running, err := w.IsContainerRunning(ctx)
	if err != nil {
		return err
	}
	if running {
		return nil // Already running
	}

	// Check if container exists but is stopped
	containers, err := w.client.ContainerList(ctx, container.ListOptions{All: true})
	if err != nil {
		return fmt.Errorf("failed to list containers: %w", err)
	}

	var containerID string
	for _, c := range containers {
		for _, name := range c.Names {
			if strings.TrimPrefix(name, "/") == w.config.ContainerName {
				containerID = c.ID
				break
			}
		}
	}

	if containerID != "" {
		// Container exists, start it
		return w.client.ContainerStart(ctx, containerID, container.StartOptions{})
	}

	// Container doesn't exist, create and start it
	return w.createAndStartContainer(ctx)
}

// createAndStartContainer creates and starts a new Whisper container.
func (w *WhisperContainer) createAndStartContainer(ctx context.Context) error {
	// Pull image if needed
	reader, err := w.client.ImagePull(ctx, w.config.Image, image.PullOptions{})
	if err != nil {
		return fmt.Errorf("failed to pull image: %w", err)
	}
	defer reader.Close()
	// Consume the pull output
	_, _ = io.Copy(io.Discard, reader)

	// Prepare port binding
	portBinding := nat.PortBinding{
		HostIP:   "0.0.0.0",
		HostPort: w.config.Port,
	}

	exposedPorts := nat.PortSet{
		"9000/tcp": struct{}{},
	}

	portMap := nat.PortMap{
		"9000/tcp": []nat.PortBinding{portBinding},
	}

	// Prepare environment variables
	env := []string{
		fmt.Sprintf("ASR_MODEL=%s", w.config.Model),
		fmt.Sprintf("ASR_DEVICE=%s", w.config.Device),
	}

	// Configure GPU support if enabled
	var hostConfig *container.HostConfig
	if w.config.UseGPU && w.config.Device != "cpu" {
		hostConfig = &container.HostConfig{
			PortBindings: portMap,
			Resources: container.Resources{
				DeviceRequests: []container.DeviceRequest{
					{
						Driver:       "nvidia",
						Count:        -1, // All GPUs
						Capabilities: [][]string{{"gpu"}},
					},
				},
			},
		}
	} else {
		hostConfig = &container.HostConfig{
			PortBindings: portMap,
		}
	}

	// Create container
	resp, err := w.client.ContainerCreate(ctx, &container.Config{
		Image:        w.config.Image,
		Env:          env,
		ExposedPorts: exposedPorts,
	}, hostConfig, nil, nil, w.config.ContainerName)
	if err != nil {
		return fmt.Errorf("failed to create container: %w", err)
	}

	// Start container
	return w.client.ContainerStart(ctx, resp.ID, container.StartOptions{})
}

// StopContainer stops the Whisper ASR container.
func (w *WhisperContainer) StopContainer(ctx context.Context) error {
	containers, err := w.client.ContainerList(ctx, container.ListOptions{})
	if err != nil {
		return fmt.Errorf("failed to list containers: %w", err)
	}

	for _, c := range containers {
		for _, name := range c.Names {
			if strings.TrimPrefix(name, "/") == w.config.ContainerName {
				timeout := int(30) // 30 seconds timeout
				return w.client.ContainerStop(ctx, c.ID, container.StopOptions{Timeout: &timeout})
			}
		}
	}
	return nil // Container not running
}

// GetContainerStatus returns the status of the Whisper container.
func (w *WhisperContainer) GetContainerStatus(ctx context.Context) (string, error) {
	containers, err := w.client.ContainerList(ctx, container.ListOptions{All: true})
	if err != nil {
		return "", fmt.Errorf("failed to list containers: %w", err)
	}

	for _, c := range containers {
		for _, name := range c.Names {
			if strings.TrimPrefix(name, "/") == w.config.ContainerName {
				return c.State, nil
			}
		}
	}
	return "unknown", nil
}

// TranscribeFile transcribes a media file using the container or fallback to external API.
func (w *WhisperContainer) TranscribeFile(ctx context.Context, filePath, language string) (*tasks.Task, error) {
	taskID := fmt.Sprintf("transcribe_%d_%s", time.Now().Unix(), uuid.NewString()[:8])

	task := tasks.Start(ctx, taskID, func(ctx context.Context) error {
		// Check if container is running
		running, err := w.IsContainerRunning(ctx)
		if err != nil {
			return fmt.Errorf("failed to check container status: %w", err)
		}

		tasks.Update(taskID, 10) // Starting transcription

		if running {
			// Use container-based transcription
			return w.transcribeWithContainer(ctx, taskID, filePath, language)
		} else {
			// Fallback to external API
			return w.transcribeWithExternalAPI(ctx, taskID, filePath, language)
		}
	})

	return task, nil
}

// Transcribe transcribes filePath against the running self-hosted Whisper
// container and returns the subtitle bytes (SRT). It speaks the ASR
// webservice's native protocol via ASRTranscribe against the mapped container
// port; it does NOT use the OpenAI API.
func (w *WhisperContainer) Transcribe(ctx context.Context, filePath, language string) ([]byte, error) {
	baseURL := fmt.Sprintf("http://localhost:%s", w.config.Port)
	return ASRTranscribe(ctx, nil, baseURL, filePath, ASROptions{Language: language, Output: "srt"})
}

// transcribeWithContainer performs transcription using the local container and
// writes the resulting SRT next to the media file (<media>.<lang>.srt).
func (w *WhisperContainer) transcribeWithContainer(ctx context.Context, taskID, filePath, language string) error {
	tasks.Update(taskID, 20) // Container available

	data, err := w.Transcribe(ctx, filePath, language)
	if err != nil {
		return fmt.Errorf("container transcription failed: %w", err)
	}

	tasks.Update(taskID, 80) // Transcription complete, writing output
	if err := writeTranscript(filePath, language, data); err != nil {
		return err
	}

	tasks.Update(taskID, 100) // Completed
	return nil
}

// transcribeWithExternalAPI performs transcription using an external
// OpenAI-compatible API and writes the resulting SRT next to the media file.
func (w *WhisperContainer) transcribeWithExternalAPI(ctx context.Context, taskID, filePath, language string) error {
	tasks.Update(taskID, 20) // Falling back to external API

	apiKey := viper.GetString("openai.api_key")
	if apiKey == "" {
		return fmt.Errorf("OpenAI API key not configured and container not available")
	}

	tasks.Update(taskID, 50) // Starting API call

	data, err := WhisperTranscribe(filePath, language, apiKey)
	if err != nil {
		return fmt.Errorf("external API transcription failed: %w", err)
	}

	tasks.Update(taskID, 80) // Transcription complete, writing output
	if err := writeTranscript(filePath, language, data); err != nil {
		return err
	}

	tasks.Update(taskID, 100) // Completed
	return nil
}

// writeTranscript writes subtitle bytes to a sidecar file next to the media,
// named <media-base>.<lang>.srt, using the same validated-path construction as
// the scanner so the output is safe and consistent with downloaded subtitles.
func writeTranscript(mediaPath, language string, data []byte) error {
	out, err := security.ValidateSubtitleOutputPath(mediaPath, language)
	if err != nil {
		return fmt.Errorf("invalid transcript output path: %w", err)
	}
	if err := os.WriteFile(out, data, 0644); err != nil {
		return fmt.Errorf("write transcript: %w", err)
	}
	return nil
}

// ValidateModel checks if the provided model is supported.
func ValidateModel(model string) bool {
	for _, m := range SupportedModels {
		if m == model {
			return true
		}
	}
	return false
}

// Close closes the Docker client connection.
func (w *WhisperContainer) Close() error {
	return w.client.Close()
}

<!-- file: sdks/go/README.md -->
<!-- version: 1.0.0 -->
<!-- guid: 550e8400-e29b-41d4-a716-446655440028 -->

# Subtitle Manager Go SDK

A comprehensive Go SDK for the Subtitle Manager API. Provides type-safe access
to all API endpoints with automatic retry, error handling, rate limiting, and
Go-idiomatic patterns.

## Features

- **Type Safety**: Complete type definitions with Go struct tags
- **Context Support**: All methods accept `context.Context` for cancellation and
  timeouts
- **Automatic Retry**: Configurable retry logic with exponential backoff
- **Rate Limiting**: Built-in rate limiting to respect API limits
- **Error Handling**: Typed errors with helper methods for common status codes
- **Pagination**: Iterator pattern for easy pagination through large result sets
- **File Operations**: Multipart file upload support for subtitle processing
- **Concurrency Safe**: Safe for use across multiple goroutines

## Installation

```bash
go get github.com/jdfalk/subtitle-manager/sdks/go
```

## Quick Start

### Basic Usage

```go
package main

import (
    "context"
    "fmt"
    "log"
    "time"

    "github.com/jdfalk/subtitle-manager/sdks/go/subtitleclient"
)

func main() {
    // Create client with default configuration
    client := subtitleclient.NewDefaultClient("http://localhost:8080", "your-api-key")

    ctx := context.Background()

    // Get system information
    systemInfo, err := client.GetSystemInfo(ctx)
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("System: %s %s\n", systemInfo.OS, systemInfo.Arch)
    fmt.Printf("Disk usage: %.1f%%\n", systemInfo.DiskUsagePercent())

    // Download subtitles
    result, err := client.DownloadSubtitles(ctx, subtitleclient.DownloadRequest{
        Path:     "/movies/example.mkv",
        Language: "en",
    })
    if err != nil {
        log.Fatal(err)
    }
    if result.Success {
        fmt.Printf("Downloaded to: %s\n", *result.SubtitlePath)
    }
}
```

### Advanced Configuration

```go
package main

import (
    "time"
    "golang.org/x/time/rate"

    "github.com/jdfalk/subtitle-manager/sdks/go/subtitleclient"
)

func main() {
    // Create client with custom configuration
    client := subtitleclient.NewClient(subtitleclient.Config{
        BaseURL:    "https://subtitles.example.com",
        APIKey:     "your-api-key",
        Timeout:    60 * time.Second,  // 60 second timeout
        MaxRetries: 5,                 // Retry up to 5 times
        UserAgent:  "MyApp/1.0.0",     // Custom user agent
        RateLimit:  rate.Limit(5),     // 5 requests per second
    })

    // Use the client...
}
```

## Authentication

### API Key Authentication

```go
// Method 1: Pass API key directly
client := subtitleclient.NewDefaultClient("http://localhost:8080", "your-api-key")

// Method 2: Use environment variable
// Set SUBTITLE_MANAGER_API_KEY environment variable
client := subtitleclient.NewClient(subtitleclient.Config{
    BaseURL: "http://localhost:8080",
    // APIKey will be read from environment variable
})
```

### Session Authentication

```go
client := subtitleclient.NewDefaultClient("http://localhost:8080", "")

// Login with username/password
loginResp, err := client.Login(ctx, "username", "password")
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Logged in as %s with role: %s\n", loginResp.Username, loginResp.Role)

// Check permissions
if loginResp.IsAdmin() {
    fmt.Println("User has admin access")
}

// Client will use session for subsequent requests
systemInfo, err := client.GetSystemInfo(ctx)
```

## File Operations

### Convert Subtitle Files

```go
package main

import (
    "context"
    "os"
    "io/ioutil"
)

func convertSubtitle(client *subtitleclient.Client, inputPath, outputPath string) error {
    ctx := context.Background()

    // Open input file
    file, err := os.Open(inputPath)
    if err != nil {
        return err
    }
    defer file.Close()

    // Convert to SRT
    srtData, err := client.ConvertSubtitle(ctx, inputPath, file)
    if err != nil {
        return err
    }

    // Save converted file
    return ioutil.WriteFile(outputPath, srtData, 0644)
}
```

### Translate Subtitles

```go
func translateSubtitle(client *subtitleclient.Client, inputPath, outputPath, language string) error {
    ctx := context.Background()

    file, err := os.Open(inputPath)
    if err != nil {
        return err
    }
    defer file.Close()

    // Translate using Google Translate
    translatedData, err := client.TranslateSubtitle(
        ctx,
        inputPath,
        file,
        language,
        subtitleclient.ProviderGoogle,
    )
    if err != nil {
        return err
    }

    return ioutil.WriteFile(outputPath, translatedData, 0644)
}
```

### Extract Embedded Subtitles

```go
func extractSubtitles(client *subtitleclient.Client, videoPath, outputPath string) error {
    ctx := context.Background()

    file, err := os.Open(videoPath)
    if err != nil {
        return err
    }
    defer file.Close()

    // Extract first English subtitle track
    subtitleData, err := client.ExtractSubtitles(ctx, videoPath, file, "en", 0)
    if err != nil {
        return err
    }

    return ioutil.WriteFile(outputPath, subtitleData, 0644)
}
```

## Library Management

### Start and Monitor Scans

```go
func monitorLibraryScan(client *subtitleclient.Client) error {
    ctx := context.Background()

    // Start scan
    scanResult, err := client.StartLibraryScan(ctx, subtitleclient.ScanRequest{
        Path:  stringPtr("/movies/new_releases"),
        Force: true,
    })
    if err != nil {
        return err
    }

    fmt.Printf("Started scan with ID: %s\n", scanResult.ScanID)

    // Monitor progress
    for {
        status, err := client.GetScanStatus(ctx)
        if err != nil {
            return err
        }

        if !status.Scanning {
            fmt.Println("Scan completed!")
            break
        }

        fmt.Printf("Progress: %.1f%%\n", status.ProgressPercent())
        if status.CurrentPath != nil {
            fmt.Printf("Current path: %s\n", *status.CurrentPath)
        }
        if status.FilesProcessed != nil && status.FilesTotal != nil {
            fmt.Printf("Files: %d/%d (remaining: %d)\n",
                *status.FilesProcessed, *status.FilesTotal, status.RemainingFiles())
        }

        time.Sleep(5 * time.Second)
    }

    return nil
}
```

### Wait for Scan Completion

```go
func waitForScanCompletion(client *subtitleclient.Client) error {
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
    defer cancel()

    // Start scan
    _, err := client.StartLibraryScan(ctx, subtitleclient.ScanRequest{Force: true})
    if err != nil {
        return err
    }

    // Wait for completion with 5 second check interval
    finalStatus, err := client.WaitForScanCompletion(ctx, 5*time.Second)
    if err != nil {
        return err
    }

    fmt.Printf("Scan completed! Progress: %.1f%%\n", finalStatus.ProgressPercent())
    return nil
}
```

## History and Monitoring

### Get Operation History

```go
func getRecentHistory(client *subtitleclient.Client) error {
    ctx := context.Background()

    // Get recent download history
    history, err := client.GetHistory(ctx, subtitleclient.HistoryParams{
        Page:      1,
        Limit:     50,
        Type:      subtitleclient.OperationTypeDownload,
        StartDate: time.Now().AddDate(0, 0, -7), // Last 7 days
    })
    if err != nil {
        return err
    }

    fmt.Printf("Found %d operations (page %d of %d)\n",
        history.Total, history.Page, history.TotalPages())

    for _, item := range history.Items {
        fmt.Printf("%s: %s - %s\n",
            item.CreatedAt.Format("2006-01-02 15:04"),
            item.Type,
            item.FilePath)

        if item.IsSuccess() && item.SubtitlePath != nil {
            fmt.Printf("  Success: %s\n", *item.SubtitlePath)
        } else if item.IsFailed() && item.ErrorMessage != nil {
            fmt.Printf("  Failed: %s\n", *item.ErrorMessage)
        }
    }

    return nil
}
```

### Iterate Through All History

```go
func iterateAllHistory(client *subtitleclient.Client) error {
    ctx := context.Background()

    // Create iterator for all download operations
    iterator := client.GetHistoryIterator(ctx, subtitleclient.HistoryParams{
        Type:  subtitleclient.OperationTypeDownload,
        Limit: 100, // Process 100 items at a time
    })

    successCount := 0
    failureCount := 0

    // Iterate through all pages
    for iterator.Next(ctx) {
        item := iterator.Item()

        if item.IsSuccess() {
            successCount++
        } else if item.IsFailed() {
            failureCount++
        }
    }

    if err := iterator.Err(); err != nil {
        return err
    }

    fmt.Printf("Total: %d, Success: %d, Failures: %d\n",
        iterator.Total(), successCount, failureCount)

    return nil
}
```

### Get Application Logs

```go
func getErrorLogs(client *subtitleclient.Client) error {
    ctx := context.Background()

    logs, err := client.GetLogs(ctx, subtitleclient.LogParams{
        Level: subtitleclient.LogLevelError,
        Limit: 100,
    })
    if err != nil {
        return err
    }

    fmt.Printf("Found %d error logs:\n", len(logs))
    for _, log := range logs {
        fmt.Printf("%s [%s] %s: %s\n",
            log.Timestamp.Format("2006-01-02 15:04:05"),
            log.Level,
            log.Component,
            log.Message)

        if len(log.Fields) > 0 {
            fmt.Printf("  Fields: %+v\n", log.Fields)
        }
    }

    return nil
}
```

## Error Handling

### Comprehensive Error Handling

```go
func handleAPIErrors(client *subtitleclient.Client) {
    ctx := context.Background()

    _, err := client.GetSystemInfo(ctx)
    if err != nil {
        if apiErr, ok := err.(*subtitleclient.APIError); ok {
            switch {
            case apiErr.IsAuthenticationError():
                fmt.Println("Authentication failed - check your API key")
            case apiErr.IsAuthorizationError():
                fmt.Println("Insufficient permissions")
            case apiErr.IsNotFoundError():
                fmt.Println("Resource not found")
            case apiErr.IsRateLimitError():
                fmt.Println("Rate limited - slow down requests")
            default:
                fmt.Printf("API error (%d): %s\n", apiErr.StatusCode, apiErr.Message)
            }
        } else {
            fmt.Printf("Network error: %v\n", err)
        }
    }
}
```

### Retry with Custom Logic

```go
func downloadWithRetry(client *subtitleclient.Client, path, language string) (*subtitleclient.DownloadResult, error) {
    ctx := context.Background()
    maxAttempts := 3

    for attempt := 1; attempt <= maxAttempts; attempt++ {
        result, err := client.DownloadSubtitles(ctx, subtitleclient.DownloadRequest{
            Path:     path,
            Language: language,
        })

        if err == nil {
            return result, nil
        }

        // Check if it's a retryable error
        if apiErr, ok := err.(*subtitleclient.APIError); ok {
            if apiErr.IsRateLimitError() && attempt < maxAttempts {
                fmt.Printf("Rate limited, retrying in %d seconds...\n", attempt*2)
                time.Sleep(time.Duration(attempt*2) * time.Second)
                continue
            }
            if apiErr.StatusCode >= 500 && attempt < maxAttempts {
                fmt.Printf("Server error, retrying attempt %d...\n", attempt+1)
                time.Sleep(time.Duration(attempt) * time.Second)
                continue
            }
        }

        return nil, err
    }

    return nil, fmt.Errorf("max retry attempts exceeded")
}
```

## Concurrent Operations

### Batch Download with Worker Pool

```go
func batchDownloadSubtitles(client *subtitleclient.Client, requests []subtitleclient.DownloadRequest) {
    const numWorkers = 5

    // Create channels
    jobs := make(chan subtitleclient.DownloadRequest, len(requests))
    results := make(chan result, len(requests))

    // Start workers
    for w := 0; w < numWorkers; w++ {
        go worker(client, jobs, results)
    }

    // Send jobs
    for _, req := range requests {
        jobs <- req
    }
    close(jobs)

    // Collect results
    for i := 0; i < len(requests); i++ {
        result := <-results
        if result.err != nil {
            fmt.Printf("Failed to download %s: %v\n", result.path, result.err)
        } else if result.downloadResult.Success {
            fmt.Printf("Downloaded %s to %s\n", result.path, *result.downloadResult.SubtitlePath)
        } else {
            fmt.Printf("No subtitles found for %s\n", result.path)
        }
    }
}

type result struct {
    path           string
    downloadResult *subtitleclient.DownloadResult
    err            error
}

func worker(client *subtitleclient.Client, jobs <-chan subtitleclient.DownloadRequest, results chan<- result) {
    ctx := context.Background()

    for req := range jobs {
        downloadResult, err := client.DownloadSubtitles(ctx, req)
        results <- result{
            path:           req.Path,
            downloadResult: downloadResult,
            err:            err,
        }
    }
}
```

### Context-Aware Operations

```go
func contextAwareOperations(client *subtitleclient.Client) {
    // Operation with timeout
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()

    systemInfo, err := client.GetSystemInfo(ctx)
    if err != nil {
        if errors.Is(err, context.DeadlineExceeded) {
            fmt.Println("Operation timed out")
        } else {
            fmt.Printf("Error: %v\n", err)
        }
        return
    }

    fmt.Printf("System: %s %s\n", systemInfo.OS, systemInfo.Arch)

    // Cancellable operation
    ctx, cancel = context.WithCancel(context.Background())

    // Cancel after 10 seconds
    go func() {
        time.Sleep(10 * time.Second)
        cancel()
    }()

    _, err = client.WaitForScanCompletion(ctx, 1*time.Second)
    if err != nil && errors.Is(err, context.Canceled) {
        fmt.Println("Operation was cancelled")
    }
}
```

## OAuth2 Management

### Admin Operations

```go
func manageOAuth(client *subtitleclient.Client) error {
    ctx := context.Background()

    // Generate new OAuth credentials (admin only)
    creds, err := client.GenerateGitHubOAuth(ctx)
    if err != nil {
        return err
    }

    fmt.Printf("Generated OAuth credentials:\n")
    fmt.Printf("Client ID: %s\n", creds.ClientID)
    fmt.Printf("Client Secret: %s\n", creds.ClientSecret)
    if creds.RedirectURL != nil {
        fmt.Printf("Redirect URL: %s\n", *creds.RedirectURL)
    }

    // Regenerate secret
    newCreds, err := client.RegenerateGitHubOAuth(ctx)
    if err != nil {
        return err
    }

    fmt.Printf("Regenerated secret: %s\n", newCreds.ClientSecret)

    return nil
}
```

## Best Practices

### Health Monitoring

```go
func healthMonitor(client *subtitleclient.Client) {
    ticker := time.NewTicker(30 * time.Second)
    defer ticker.Stop()

    for range ticker.C {
        ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)

        if client.HealthCheck(ctx) {
            fmt.Println("API is healthy")
        } else {
            fmt.Println("API health check failed")
        }

        cancel()
    }
}
```

### Resource Management

```go
func processFiles(client *subtitleclient.Client, filePaths []string) error {
    ctx := context.Background()

    for _, path := range filePaths {
        func() {
            // Open file
            file, err := os.Open(path)
            if err != nil {
                fmt.Printf("Failed to open %s: %v\n", path, err)
                return
            }
            defer file.Close() // Ensure file is closed

            // Process file
            data, err := client.ConvertSubtitle(ctx, path, file)
            if err != nil {
                fmt.Printf("Failed to convert %s: %v\n", path, err)
                return
            }

            // Save result
            outputPath := strings.TrimSuffix(path, filepath.Ext(path)) + ".srt"
            if err := ioutil.WriteFile(outputPath, data, 0644); err != nil {
                fmt.Printf("Failed to save %s: %v\n", outputPath, err)
                return
            }

            fmt.Printf("Converted %s to %s\n", path, outputPath)
        }()
    }

    return nil
}
```

## Testing

### Unit Tests

```bash
# Run tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Run tests with race detection
go test -race ./...

# Verbose output
go test -v ./...
```

### Integration Tests

```go
func TestIntegration(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping integration test in short mode")
    }

    client := subtitleclient.NewDefaultClient("http://localhost:8080", "test-api-key")
    ctx := context.Background()

    // Test health check
    healthy := client.HealthCheck(ctx)
    assert.True(t, healthy, "API should be healthy")

    // Test system info
    systemInfo, err := client.GetSystemInfo(ctx)
    require.NoError(t, err)
    assert.NotEmpty(t, systemInfo.GoVersion)
}
```

## Examples

Complete examples can be found in the `examples/` directory:

- [Basic Operations](examples/basic/main.go)
- [File Processing](examples/files/main.go)
- [Batch Operations](examples/batch/main.go)
- [Monitoring](examples/monitoring/main.go)

## Documentation

- [API Reference](https://pkg.go.dev/github.com/jdfalk/subtitle-manager/sdks/go/subtitleclient)
- [Examples](examples/)
- [Contributing](../../CONTRIBUTING.md)

## Support

- **Issues**: [GitHub Issues](https://github.com/jdfalk/subtitle-manager/issues)
- **Documentation**:
  [API Documentation](https://github.com/jdfalk/subtitle-manager/tree/main/docs/api)
- **Source Code**:
  [GitHub Repository](https://github.com/jdfalk/subtitle-manager)

## License

This project is licensed under the PolyForm Noncommercial License 1.0.0 -
free for noncommercial use; commercial use requires a separate license from
the author. See the [LICENSE](../../LICENSE) file for details.

## Helper Functions

```go
// Helper functions for creating pointers to basic types
func stringPtr(s string) *string {
    return &s
}

func intPtr(i int) *int {
    return &i
}

func timePtr(t time.Time) *time.Time {
    return &t
}
```

<!-- file: sdks/python/README.md -->
<!-- version: 1.0.0 -->
<!-- guid: 550e8400-e29b-41d4-a716-446655440011 -->

# Subtitle Manager Python SDK

A comprehensive Python SDK for the Subtitle Manager API. Provides type-safe
access to all API endpoints with automatic retry, error handling, and pagination
support.

## Features

- **Type Safety**: Full type hints and data classes for all API responses
- **Authentication**: Support for API keys, session cookies, and OAuth2
- **Error Handling**: Specific exception types for different error conditions
- **Automatic Retry**: Configurable retry logic with exponential backoff
- **Pagination**: Easy iteration over paginated results
- **File Operations**: Upload and download subtitle files with ease
- **Rate Limiting**: Automatic handling of rate limit responses

## Installation

```bash
pip install subtitle-manager-sdk
```

For development:

```bash
pip install subtitle-manager-sdk[dev]
```

## Quick Start

### Basic Usage

```python
from subtitle_manager_sdk import SubtitleManagerClient

# Initialize client with API key
client = SubtitleManagerClient(
    base_url="http://localhost:8080",
    api_key="your-api-key"
)

# Get system information
system_info = client.get_system_info()
print(f"System: {system_info.os} {system_info.arch}")

# Download subtitles
result = client.download_subtitles(
    path="/movies/example.mkv",
    language="en"
)
if result.success:
    print(f"Downloaded to: {result.subtitle_path}")
```

### Authentication

#### API Key Authentication

```python
client = SubtitleManagerClient(
    base_url="http://localhost:8080",
    api_key="your-api-key"
)
```

#### Session Authentication

```python
client = SubtitleManagerClient(base_url="http://localhost:8080")

# Login with username/password
login_response = client.login("username", "password")
print(f"Logged in as: {login_response.username}")

# Client will use session cookie for subsequent requests
system_info = client.get_system_info()
```

#### Environment Variables

```python
# Set SUBTITLE_MANAGER_API_KEY environment variable
import os
os.environ["SUBTITLE_MANAGER_API_KEY"] = "your-api-key"

# Client will automatically use the environment variable
client = SubtitleManagerClient("http://localhost:8080")
```

### File Operations

#### Convert Subtitle Format

```python
# Convert from file path
srt_content = client.convert_subtitle("/path/to/subtitle.vtt")
with open("converted.srt", "wb") as f:
    f.write(srt_content)

# Convert from file object
from io import BytesIO
vtt_data = BytesIO(b"WEBVTT\n\n00:01.000 --> 00:03.000\nHello World")
srt_content = client.convert_subtitle(vtt_data, filename="input.vtt")
```

#### Translate Subtitles

```python
from subtitle_manager_sdk.models import TranslationProvider

# Translate to Spanish using Google Translate
translated = client.translate_subtitle(
    file="/path/to/english.srt",
    language="es",
    provider=TranslationProvider.GOOGLE
)

with open("spanish.srt", "wb") as f:
    f.write(translated)
```

#### Extract Embedded Subtitles

```python
# Extract first subtitle track
subtitles = client.extract_subtitles("/path/to/video.mkv")

# Extract specific language and track
subtitles = client.extract_subtitles(
    file="/path/to/video.mkv",
    language="en",
    track=1
)
```

### Library Management

#### Start Library Scan

```python
# Scan entire library
scan_result = client.start_library_scan()
print(f"Scan started with ID: {scan_result.scan_id}")

# Scan specific path
scan_result = client.start_library_scan(
    path="/movies/new_releases",
    force=True  # Force rescan of existing files
)
```

#### Monitor Scan Progress

```python
# Get current scan status
status = client.get_scan_status()
if status.scanning:
    print(f"Scanning: {status.progress:.1%} complete")
    print(f"Current path: {status.current_path}")

# Wait for scan completion
final_status = client.wait_for_scan_completion(
    poll_interval=5,  # Check every 5 seconds
    timeout=3600      # Maximum 1 hour
)
print("Scan completed!")
```

### History and Monitoring

#### View Operation History

```python
from subtitle_manager_sdk.models import OperationType
from datetime import datetime, timedelta

# Get recent download history
history = client.get_history(
    operation_type=OperationType.DOWNLOAD,
    start_date=datetime.now() - timedelta(days=7),
    limit=50
)

for item in history.items:
    print(f"{item.created_at}: {item.type} - {item.file_path}")
    if item.status == "success":
        print(f"  Success: {item.subtitle_path}")
    else:
        print(f"  Failed: {item.error_message}")
```

#### View Application Logs

```python
from subtitle_manager_sdk.models import LogLevel

# Get recent error logs
logs = client.get_logs(level=LogLevel.ERROR, limit=100)
for log in logs:
    print(f"{log.timestamp} [{log.level}] {log.component}: {log.message}")
```

### Error Handling

```python
from subtitle_manager_sdk.exceptions import (
    AuthenticationError,
    AuthorizationError,
    RateLimitError,
    NotFoundError,
    APIError
)

try:
    result = client.download_subtitles("/path/to/movie.mkv", "en")
except AuthenticationError:
    print("Authentication failed - check your API key")
except AuthorizationError:
    print("Insufficient permissions")
except RateLimitError as e:
    print(f"Rate limited - retry after {e.retry_after} seconds")
except NotFoundError:
    print("Movie file not found")
except APIError as e:
    print(f"API error: {e.message}")
```

### Advanced Usage

#### Custom Configuration

```python
client = SubtitleManagerClient(
    base_url="https://subtitles.example.com",
    api_key="your-api-key",
    timeout=60,           # Request timeout in seconds
    max_retries=5,        # Maximum retry attempts
    retry_backoff=2.0,    # Exponential backoff factor
    user_agent="MyApp/1.0"
)
```

#### Context Manager

```python
# Automatically close session when done
with SubtitleManagerClient("http://localhost:8080", api_key="key") as client:
    system_info = client.get_system_info()
    # Session automatically closed
```

#### Health Checks

```python
# Check if API is accessible
if client.health_check():
    print("API is healthy")
else:
    print("API is not responding")
```

## Data Models

The SDK provides type-safe data models for all API responses:

### SystemInfo

```python
@dataclass
class SystemInfo:
    go_version: str
    os: str
    arch: str
    goroutines: int
    disk_free: int
    disk_total: int
    memory_usage: Optional[int] = None
    uptime: Optional[str] = None
    version: Optional[str] = None
```

### HistoryItem

```python
@dataclass
class HistoryItem:
    id: int
    type: OperationType  # download, convert, translate, extract
    file_path: str
    subtitle_path: Optional[str]
    language: Optional[str]
    provider: Optional[str]
    status: OperationStatus  # success, failed, pending
    created_at: datetime
    user_id: int
    error_message: Optional[str] = None
```

### ScanStatus

```python
@dataclass
class ScanStatus:
    scanning: bool
    progress: float  # 0.0 to 1.0
    current_path: Optional[str] = None
    files_processed: Optional[int] = None
    files_total: Optional[int] = None
    start_time: Optional[datetime] = None
    estimated_completion: Optional[datetime] = None
```

## Exception Hierarchy

```
SubtitleManagerError
├── AuthenticationError (401)
├── AuthorizationError (403)
├── NotFoundError (404)
├── RateLimitError (429)
├── APIError (4xx, 5xx)
├── ValidationError
├── NetworkError
└── TimeoutError
```

## Configuration

### Environment Variables

- `SUBTITLE_MANAGER_API_KEY`: Default API key
- `SUBTITLE_MANAGER_BASE_URL`: Default base URL

### Logging

The SDK uses Python's standard logging module:

```python
import logging

# Enable debug logging for the SDK
logging.getLogger('subtitle_manager_sdk').setLevel(logging.DEBUG)
```

## Development

### Running Tests

```bash
# Install development dependencies
pip install -e .[dev]

# Run tests
pytest

# Run tests with coverage
pytest --cov=subtitle_manager_sdk

# Run specific test file
pytest tests/test_client.py
```

### Code Formatting

```bash
# Format code
black subtitle_manager_sdk tests

# Check formatting
black --check subtitle_manager_sdk tests

# Lint code
flake8 subtitle_manager_sdk tests

# Type checking
mypy subtitle_manager_sdk
```

## Examples

See the [examples directory](examples/) for complete usage examples:

- [Basic Operations](examples/basic_operations.py)
- [File Processing](examples/file_processing.py)
- [Library Management](examples/library_management.py)
- [Error Handling](examples/error_handling.py)

## Support

- **Documentation**:
  [API Documentation](https://github.com/jdfalk/subtitle-manager/tree/main/docs/api)
- **Issues**: [GitHub Issues](https://github.com/jdfalk/subtitle-manager/issues)
- **Source Code**:
  [GitHub Repository](https://github.com/jdfalk/subtitle-manager)

## License

This project is licensed under the PolyForm Noncommercial License 1.0.0 -
free for noncommercial use; commercial use requires a separate license from
the author. See the [LICENSE](../../LICENSE) file for details.

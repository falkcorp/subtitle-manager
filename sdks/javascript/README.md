<!-- file: sdks/javascript/README.md -->
<!-- version: 1.0.0 -->
<!-- guid: 550e8400-e29b-41d4-a716-446655440020 -->

# Subtitle Manager JavaScript/TypeScript SDK

A comprehensive JavaScript/TypeScript SDK for the Subtitle Manager API. Provides
full type safety, automatic retry, error handling, and modern async/await
patterns for both Node.js and browser environments.

## Features

- **Full TypeScript Support**: Complete type definitions and IntelliSense
  support
- **Cross-Platform**: Works in Node.js, browsers, and React Native
- **Modern Async/Await**: Promise-based API with async generators for pagination
- **Automatic Retry**: Configurable retry logic with exponential backoff
- **Error Handling**: Specific error types for different HTTP status codes
- **File Operations**: Upload and download subtitle files with progress tracking
- **Authentication**: Support for API keys, session cookies, and OAuth2
- **Rate Limiting**: Automatic handling of rate limit responses

## Installation

```bash
npm install subtitle-manager-sdk
```

For TypeScript projects:

```bash
npm install subtitle-manager-sdk
npm install --save-dev @types/node  # For Node.js types
```

## Quick Start

### Basic Usage (JavaScript)

```javascript
const { SubtitleManagerClient } = require('subtitle-manager-sdk');

// Initialize client
const client = new SubtitleManagerClient({
  baseURL: 'http://localhost:8080',
  apiKey: 'your-api-key',
});

// Get system information
const systemInfo = await client.getSystemInfo();
console.log(`System: ${systemInfo.os} ${systemInfo.arch}`);

// Download subtitles
const result = await client.downloadSubtitles('/movies/example.mkv', 'en');
if (result.success) {
  console.log(`Downloaded to: ${result.subtitle_path}`);
}
```

### TypeScript Usage

```typescript
import {
  SubtitleManagerClient,
  OperationType,
  TranslationProvider,
} from 'subtitle-manager-sdk';

const client = new SubtitleManagerClient({
  baseURL: 'http://localhost:8080',
  apiKey: process.env.SUBTITLE_MANAGER_API_KEY!,
  timeout: 60000,
  maxRetries: 5,
});

// Type-safe operations
const history = await client.getHistory({
  type: OperationType.DOWNLOAD,
  limit: 50,
});

// Process each history item with full type safety
history.items.forEach(item => {
  console.log(`${item.type}: ${item.file_path} - ${item.status}`);
});
```

## Authentication

### API Key Authentication

```javascript
const client = new SubtitleManagerClient({
  baseURL: 'http://localhost:8080',
  apiKey: 'your-api-key',
});
```

### Session Authentication

```javascript
const client = new SubtitleManagerClient({
  baseURL: 'http://localhost:8080',
});

// Login with username/password
const loginResponse = await client.login('username', 'password');
console.log(`Logged in as: ${loginResponse.username}`);

// Client automatically uses session cookie for subsequent requests
const systemInfo = await client.getSystemInfo();
```

### Environment Variables

```javascript
// Set SUBTITLE_MANAGER_API_KEY environment variable
process.env.SUBTITLE_MANAGER_API_KEY = 'your-api-key';

// Client automatically uses environment variable
const client = new SubtitleManagerClient({
  baseURL: 'http://localhost:8080',
});
```

### Dynamic Authentication

```javascript
const client = new SubtitleManagerClient({
  baseURL: 'http://localhost:8080',
});

// Set API key dynamically
client.setApiKey('new-api-key');

// Set session cookie
client.setSessionCookie('session-cookie-value');

// Clear all authentication
client.clearAuth();
```

## File Operations

### Convert Subtitle Format

```javascript
// Browser environment with file input
const fileInput = document.getElementById('subtitle-file');
const file = fileInput.files[0];

const convertedBlob = await client.convertSubtitle(file);

// Download converted file
const url = URL.createObjectURL(convertedBlob);
const a = document.createElement('a');
a.href = url;
a.download = 'converted.srt';
a.click();
```

### Translate Subtitles

```javascript
import { TranslationProvider } from 'subtitle-manager-sdk';

// Translate to Spanish using Google Translate
const translatedBlob = await client.translateSubtitle(
  file,
  'es',
  TranslationProvider.GOOGLE
);

// Save translated file (Node.js)
import fs from 'fs';
const arrayBuffer = await translatedBlob.arrayBuffer();
fs.writeFileSync('spanish.srt', Buffer.from(arrayBuffer));
```

### Extract Embedded Subtitles

```javascript
// Extract first subtitle track
const subtitles = await client.extractSubtitles(videoFile);

// Extract specific language and track
const subtitles = await client.extractSubtitles(
  videoFile,
  'en', // language
  1 // track number
);
```

### Batch File Operations

```javascript
// Upload and convert multiple files
const files = [...fileInputElement.files];
const convertedFiles = await Promise.all(
  files.map(file => client.convertSubtitle(file))
);

// Download subtitles for multiple movies
const moviePaths = ['/movies/movie1.mkv', '/movies/movie2.mkv'];
const downloads = [];

for await (const { index, result, error } of client.downloadMultipleSubtitles(
  moviePaths.map(path => ({ path, language: 'en' }))
)) {
  if (error) {
    console.error(`Failed to download ${moviePaths[index]}:`, error.message);
  } else {
    console.log(`Downloaded: ${result.subtitle_path}`);
    downloads.push(result);
  }
}
```

## Library Management

### Start and Monitor Scans

```javascript
// Start library scan
const scanResult = await client.startLibraryScan('/movies/new_releases', true);
console.log(`Scan started: ${scanResult.scan_id}`);

// Monitor scan progress
const status = await client.getScanStatus();
if (status.scanning) {
  console.log(`Progress: ${status.progressPercent}%`);
  console.log(`Current path: ${status.current_path}`);
  console.log(
    `Files processed: ${status.files_processed}/${status.files_total}`
  );
}

// Wait for completion
try {
  const finalStatus = await client.waitForScanCompletion(5000, 3600000); // 5s interval, 1h timeout
  console.log('Scan completed!');
} catch (error) {
  console.error('Scan timed out:', error.message);
}
```

### Real-time Progress Updates

```javascript
// Monitor scan progress with real-time updates
async function monitorScan() {
  const scanResult = await client.startLibraryScan();

  const progressInterval = setInterval(async () => {
    try {
      const status = await client.getScanStatus();

      if (status.scanning) {
        console.log(`Progress: ${status.progressPercent}%`);
        if (status.estimated_completion) {
          const eta = new Date(status.estimated_completion);
          console.log(`ETA: ${eta.toLocaleTimeString()}`);
        }
      } else {
        console.log('Scan completed!');
        clearInterval(progressInterval);
      }
    } catch (error) {
      console.error('Error checking scan status:', error);
      clearInterval(progressInterval);
    }
  }, 5000);
}
```

## History and Monitoring

### Pagination with Async Generators

```javascript
import { OperationType } from 'subtitle-manager-sdk';

// Iterate through all download history
for await (const historyPage of client.getHistoryPages({
  type: OperationType.DOWNLOAD,
  limit: 50,
})) {
  historyPage.forEach(item => {
    console.log(`${item.createdAtDate}: ${item.file_path}`);
    if (item.isSuccess) {
      console.log(`  Success: ${item.subtitle_path}`);
    } else if (item.isFailed) {
      console.log(`  Failed: ${item.error_message}`);
    }
  });
}
```

### Advanced History Filtering

```javascript
// Get recent failed operations
const oneWeekAgo = new Date();
oneWeekAgo.setDate(oneWeekAgo.getDate() - 7);

const recentHistory = await client.getHistory({
  start_date: oneWeekAgo.toISOString(),
  limit: 100,
});

const failures = recentHistory.items.filter(item => item.isFailed);
console.log(`Recent failures: ${failures.length}`);

// Group by error type
const errorGroups = failures.reduce((groups, item) => {
  const error = item.error_message || 'Unknown error';
  groups[error] = (groups[error] || 0) + 1;
  return groups;
}, {});

console.log('Error breakdown:', errorGroups);
```

### Application Logs

```javascript
import { LogLevel } from 'subtitle-manager-sdk';

// Get recent error logs
const errorLogs = await client.getLogs({
  level: LogLevel.ERROR,
  limit: 100,
});

errorLogs.forEach(log => {
  console.log(
    `${log.timestampDate.toISOString()} [${log.level}] ${log.component}: ${log.message}`
  );
  if (Object.keys(log.fields).length > 0) {
    console.log('  Fields:', log.fields);
  }
});
```

## Error Handling

### Comprehensive Error Handling

```javascript
import {
  AuthenticationError,
  AuthorizationError,
  RateLimitError,
  NotFoundError,
  ValidationError,
  NetworkError,
  TimeoutError,
  APIError,
} from 'subtitle-manager-sdk';

try {
  const result = await client.downloadSubtitles('/path/to/movie.mkv', 'en');
  console.log('Download successful:', result);
} catch (error) {
  if (error instanceof AuthenticationError) {
    console.error('Authentication failed - check your API key');
  } else if (error instanceof AuthorizationError) {
    console.error('Insufficient permissions');
  } else if (error instanceof RateLimitError) {
    console.error(`Rate limited - retry after ${error.retryAfter} seconds`);
    // Implement exponential backoff
    setTimeout(() => {
      // Retry operation
    }, error.retryAfter * 1000);
  } else if (error instanceof NotFoundError) {
    console.error('Movie file not found');
  } else if (error instanceof ValidationError) {
    console.error('Invalid request data:', error.details);
  } else if (error instanceof NetworkError) {
    console.error('Network error:', error.message);
  } else if (error instanceof TimeoutError) {
    console.error('Request timed out');
  } else if (error instanceof APIError) {
    console.error(`API error (${error.statusCode}):`, error.message);
  } else {
    console.error('Unexpected error:', error);
  }
}
```

### Retry with Exponential Backoff

```javascript
async function downloadWithRetry(path, language, maxRetries = 3) {
  for (let attempt = 1; attempt <= maxRetries; attempt++) {
    try {
      return await client.downloadSubtitles(path, language);
    } catch (error) {
      if (error instanceof RateLimitError && attempt < maxRetries) {
        const delay = error.retryAfter
          ? error.retryAfter * 1000
          : Math.pow(2, attempt) * 1000;
        console.log(
          `Rate limited, waiting ${delay}ms before retry ${attempt + 1}`
        );
        await new Promise(resolve => setTimeout(resolve, delay));
        continue;
      }
      throw error;
    }
  }
}
```

## Browser Usage

### Using with React

```jsx
import React, { useState, useEffect } from 'react';
import { SubtitleManagerClient } from 'subtitle-manager-sdk';

const SubtitleUploader = () => {
  const [client] = useState(
    () =>
      new SubtitleManagerClient({
        baseURL: process.env.REACT_APP_API_URL,
        apiKey: process.env.REACT_APP_API_KEY,
      })
  );

  const [file, setFile] = useState(null);
  const [converting, setConverting] = useState(false);

  const handleConvert = async () => {
    if (!file) return;

    setConverting(true);
    try {
      const convertedBlob = await client.convertSubtitle(file);

      // Download converted file
      const url = URL.createObjectURL(convertedBlob);
      const a = document.createElement('a');
      a.href = url;
      a.download = `converted-${file.name}`;
      a.click();
      URL.revokeObjectURL(url);
    } catch (error) {
      console.error('Conversion failed:', error);
    } finally {
      setConverting(false);
    }
  };

  return (
    <div>
      <input
        type="file"
        accept=".vtt,.ass,.ssa,.smi"
        onChange={e => setFile(e.target.files[0])}
      />
      <button onClick={handleConvert} disabled={!file || converting}>
        {converting ? 'Converting...' : 'Convert to SRT'}
      </button>
    </div>
  );
};
```

### Web Workers

```javascript
// main.js
import { SubtitleManagerClient } from 'subtitle-manager-sdk';

const worker = new Worker('subtitle-worker.js');

worker.postMessage({
  type: 'convert',
  file: selectedFile,
  config: {
    baseURL: 'http://localhost:8080',
    apiKey: 'your-api-key',
  },
});

worker.onmessage = event => {
  const { type, result, error } = event.data;
  if (type === 'convert-complete') {
    console.log('Conversion completed:', result);
  } else if (type === 'error') {
    console.error('Worker error:', error);
  }
};
```

```javascript
// subtitle-worker.js
import { SubtitleManagerClient } from 'subtitle-manager-sdk';

self.onmessage = async event => {
  const { type, file, config } = event.data;

  if (type === 'convert') {
    try {
      const client = new SubtitleManagerClient(config);
      const result = await client.convertSubtitle(file);

      self.postMessage({
        type: 'convert-complete',
        result: result,
      });
    } catch (error) {
      self.postMessage({
        type: 'error',
        error: error.message,
      });
    }
  }
};
```

## Node.js Usage

### File System Integration

```javascript
import fs from 'fs';
import path from 'path';
import { SubtitleManagerClient } from 'subtitle-manager-sdk';

const client = new SubtitleManagerClient({
  baseURL: 'http://localhost:8080',
  apiKey: process.env.SUBTITLE_MANAGER_API_KEY,
});

// Convert local subtitle files
async function convertLocalFile(inputPath, outputPath) {
  const fileBuffer = fs.readFileSync(inputPath);
  const file = new Blob([fileBuffer], { type: 'application/octet-stream' });
  const filename = path.basename(inputPath);

  const convertedBlob = await client.convertSubtitle(file, filename);
  const arrayBuffer = await convertedBlob.arrayBuffer();

  fs.writeFileSync(outputPath, Buffer.from(arrayBuffer));
  console.log(`Converted ${inputPath} to ${outputPath}`);
}

// Batch convert all subtitle files in directory
async function convertDirectory(inputDir, outputDir) {
  const files = fs.readdirSync(inputDir);
  const subtitleFiles = files.filter(file =>
    ['.vtt', '.ass', '.ssa', '.smi'].includes(path.extname(file).toLowerCase())
  );

  for (const file of subtitleFiles) {
    const inputPath = path.join(inputDir, file);
    const outputPath = path.join(
      outputDir,
      path.basename(file, path.extname(file)) + '.srt'
    );

    try {
      await convertLocalFile(inputPath, outputPath);
    } catch (error) {
      console.error(`Failed to convert ${file}:`, error.message);
    }
  }
}
```

## Configuration

### Client Configuration Options

```typescript
interface ClientConfig {
  baseURL: string; // API base URL
  apiKey?: string; // API key for authentication
  sessionCookie?: string; // Session cookie value
  timeout?: number; // Request timeout in milliseconds (default: 30000)
  maxRetries?: number; // Maximum retry attempts (default: 3)
  retryDelay?: number; // Base retry delay in milliseconds (default: 1000)
  userAgent?: string; // Custom user agent string
  debug?: boolean; // Enable debug logging (default: false)
}
```

### Advanced Configuration

```javascript
const client = new SubtitleManagerClient({
  baseURL: 'https://subtitles.example.com',
  apiKey: process.env.API_KEY,
  timeout: 120000, // 2 minutes
  maxRetries: 5, // More aggressive retrying
  retryDelay: 2000, // 2 second base delay
  userAgent: 'MyApp/1.0.0', // Custom user agent
  debug: true, // Enable request logging
});
```

## Type Definitions

The SDK provides comprehensive TypeScript definitions for all API operations:

```typescript
// System information
interface SystemInfo {
  go_version: string;
  os: string;
  arch: string;
  goroutines: number;
  disk_free: number;
  disk_total: number;
  memory_usage?: number;
  uptime?: string;
  version?: string;
}

// Operation history
interface HistoryItem {
  id: number;
  type: OperationType;
  file_path: string;
  subtitle_path?: string;
  language?: string;
  provider?: string;
  status: OperationStatus;
  created_at: string;
  user_id: number;
  error_message?: string;
}

// Enums for type safety
enum OperationType {
  DOWNLOAD = 'download',
  CONVERT = 'convert',
  TRANSLATE = 'translate',
  EXTRACT = 'extract',
}

enum OperationStatus {
  SUCCESS = 'success',
  FAILED = 'failed',
  PENDING = 'pending',
}
```

## Development

### Building from Source

```bash
# Clone repository
git clone https://github.com/jdfalk/subtitle-manager.git
cd subtitle-manager/sdks/javascript

# Install dependencies
npm install

# Build the SDK
npm run build

# Run tests
npm test

# Run tests with coverage
npm run test:coverage

# Lint code
npm run lint
```

### Testing

```bash
# Run all tests
npm test

# Run tests in watch mode
npm run test:watch

# Generate coverage report
npm run test:coverage
```

## Examples

### Complete Subtitle Processing Pipeline

```javascript
import {
  SubtitleManagerClient,
  TranslationProvider,
} from 'subtitle-manager-sdk';

class SubtitleProcessor {
  constructor(apiKey) {
    this.client = new SubtitleManagerClient({
      baseURL: 'http://localhost:8080',
      apiKey: apiKey,
      debug: true,
    });
  }

  async processSubtitle(file, targetLanguage) {
    try {
      // Step 1: Convert to SRT format
      console.log('Converting to SRT...');
      const srtBlob = await this.client.convertSubtitle(file);

      // Step 2: Translate if needed
      if (targetLanguage) {
        console.log(`Translating to ${targetLanguage}...`);
        const translatedBlob = await this.client.translateSubtitle(
          srtBlob,
          targetLanguage,
          TranslationProvider.GOOGLE
        );
        return translatedBlob;
      }

      return srtBlob;
    } catch (error) {
      console.error('Processing failed:', error);
      throw error;
    }
  }

  async downloadForMovie(moviePath, languages = ['en']) {
    const results = [];

    for (const language of languages) {
      try {
        const result = await this.client.downloadSubtitles(moviePath, language);
        results.push({ language, result });
      } catch (error) {
        results.push({ language, error: error.message });
      }
    }

    return results;
  }
}

// Usage
const processor = new SubtitleProcessor('your-api-key');

// Process uploaded file
const processedSubtitle = await processor.processSubtitle(uploadedFile, 'es');

// Download subtitles for movie
const downloadResults = await processor.downloadForMovie(
  '/movies/example.mkv',
  ['en', 'es', 'fr']
);
```

## License

This project is licensed under the PolyForm Noncommercial License 1.0.0 -
free for noncommercial use; commercial use requires a separate license from
the author. See the [LICENSE](../../LICENSE) file for details.

## Support

- **Documentation**:
  [API Documentation](https://github.com/jdfalk/subtitle-manager/tree/main/docs/api)
- **Issues**: [GitHub Issues](https://github.com/jdfalk/subtitle-manager/issues)
- **Source Code**:
  [GitHub Repository](https://github.com/jdfalk/subtitle-manager)

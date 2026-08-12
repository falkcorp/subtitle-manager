<!-- file: docs/MONITORING.md -->
<!-- version: 1.1.0 -->
<!-- last-edited: 2026-08-12 -->
<!-- guid: d4e5f6g7-h8i9-0j1k-2l3m-n4o5p6q7r8s9 -->

# Monitoring and Metrics

Subtitle Manager exposes Prometheus metrics for monitoring and observability.

## Prerequisites

The web server stores authentication in SQLite, which is compiled into every
build via the pure-Go `modernc.org/sqlite` driver. No build tag and no CGO:

```bash
CGO_ENABLED=0 go build -o subtitle-manager .
```

## Prometheus Metrics Endpoint

The application exposes a `/metrics` endpoint that provides metrics in
Prometheus format.

### Endpoint Details

- **URL**: `http://localhost:8080/metrics` (or your configured address)
- **Authentication**: Not required (standard practice for monitoring endpoints)
- **Content-Type**: `text/plain; version=0.0.4; charset=utf-8`

## Available Metrics

### Provider Metrics

#### `subtitle_manager_provider_requests_total`

Counter tracking requests made to subtitle providers.

**Labels:**

- `provider`: Name of the subtitle provider (e.g., "opensubtitles", "subscene")
- `status`: Request status ("success" or "error")

**Example:**

```
subtitle_manager_provider_requests_total{provider="opensubtitles",status="success"} 42
subtitle_manager_provider_requests_total{provider="opensubtitles",status="error"} 3
```

#### `subtitle_manager_subtitle_downloads_total`

Counter tracking successful subtitle downloads.

**Labels:**

- `provider`: Name of the subtitle provider
- `language`: Language code of downloaded subtitles (e.g., "en", "fr", "es")

**Example:**

```
subtitle_manager_subtitle_downloads_total{provider="opensubtitles",language="en"} 25
subtitle_manager_subtitle_downloads_total{provider="subscene",language="fr"} 8
```

### Translation Metrics

#### `subtitle_manager_translation_requests_total`

Counter tracking translation requests processed.

**Labels:**

- `service`: Translation service used ("google", "openai", "grpc")
- `target_language`: Target language for translation
- `status`: Request status ("success", "error", "bad_request")

**Example:**

```
subtitle_manager_translation_requests_total{service="google",target_language="en",status="success"} 15
subtitle_manager_translation_requests_total{service="openai",target_language="fr",status="error"} 2
```

### API Metrics

#### `subtitle_manager_api_requests_total`

Counter tracking API requests by endpoint.

**Labels:**

- `endpoint`: API endpoint path (e.g., "/api/download", "/api/translate")
- `method`: HTTP method ("GET", "POST", "PUT", "DELETE")
- `status_code`: HTTP status code ("200", "400", "500", etc.)

**Example:**

```
subtitle_manager_api_requests_total{endpoint="/api/download",method="POST",status_code="200"} 18
subtitle_manager_api_requests_total{endpoint="/api/translate",method="POST",status_code="400"} 5
```

#### `subtitle_manager_request_duration_seconds`

Histogram tracking API request duration.

**Labels:**

- `endpoint`: API endpoint path
- `method`: HTTP method

**Buckets:** Uses Prometheus default buckets (0.005, 0.01, 0.025, 0.05, 0.1,
0.25, 0.5, 1, 2.5, 5, 10)

### System Metrics

#### `subtitle_manager_active_sessions`

Gauge tracking the number of active user sessions.

**Example:**

```
subtitle_manager_active_sessions 3
```

## Prometheus Configuration

### Basic Prometheus Configuration

Add the following job to your `prometheus.yml`:

```yaml
scrape_configs:
  - job_name: 'subtitle-manager'
    static_configs:
      - targets: ['localhost:8080']
    metrics_path: '/metrics'
    scrape_interval: 30s
```

### With Custom Base URL

If you're running Subtitle Manager with a custom base URL (e.g.,
`--base-url=/subtitles`):

```yaml
scrape_configs:
  - job_name: 'subtitle-manager'
    static_configs:
      - targets: ['localhost:8080']
    metrics_path: '/subtitles/metrics'
    scrape_interval: 30s
```

### Docker Compose Example

```yaml
version: '3.8'
services:
  subtitle-manager:
    image: ghcr.io/jdfalk/subtitle-manager:latest
    ports:
      - '8080:8080'
    volumes:
      - ./config:/config
      - ./downloads:/downloads
    environment:
      - DB_BACKEND=pebble

  prometheus:
    image: prom/prometheus:latest
    ports:
      - '9090:9090'
    volumes:
      - ./prometheus.yml:/etc/prometheus/prometheus.yml
    command:
      - '--config.file=/etc/prometheus/prometheus.yml'
      - '--storage.tsdb.path=/prometheus'
      - '--web.console.libraries=/etc/prometheus/console_libraries'
      - '--web.console.templates=/etc/prometheus/consoles'
```

## Grafana Dashboard

### Key Metrics to Monitor

1. **Request Rate**: Rate of subtitle downloads and translations
2. **Error Rate**: Percentage of failed requests by provider/service
3. **Response Time**: 95th percentile response times for API endpoints
4. **Provider Performance**: Success rates and response times per provider
5. **Active Users**: Number of active sessions

### Example Queries

**Request Rate (requests per minute):**

```promql
rate(subtitle_manager_provider_requests_total[5m]) * 60
```

**Error Rate by Provider:**

```promql
rate(subtitle_manager_provider_requests_total{status="error"}[5m])
/
rate(subtitle_manager_provider_requests_total[5m])
```

**95th Percentile Response Time:**

```promql
histogram_quantile(0.95, rate(subtitle_manager_request_duration_seconds_bucket[5m]))
```

**Downloads by Language:**

```promql
sum by(language) (rate(subtitle_manager_subtitle_downloads_total[5m]))
```

## Alerting

### Example Alerting Rules

```yaml
groups:
  - name: subtitle-manager
    rules:
      - alert: SubtitleManagerHighErrorRate
        expr:
          rate(subtitle_manager_provider_requests_total{status="error"}[5m]) /
          rate(subtitle_manager_provider_requests_total[5m]) > 0.1
        for: 5m
        labels:
          severity: warning
        annotations:
          summary:
            'High error rate for subtitle provider {{ $labels.provider }}'
          description:
            'Error rate for provider {{ $labels.provider }} is {{ $value |
            humanizePercentage }}'

      - alert: SubtitleManagerDown
        expr: up{job="subtitle-manager"} == 0
        for: 1m
        labels:
          severity: critical
        annotations:
          summary: 'Subtitle Manager is down'
          description: 'Subtitle Manager has been down for more than 1 minute'
```

## Security Considerations

- The `/metrics` endpoint does not require authentication, which is standard
  practice for monitoring endpoints
- Ensure your monitoring infrastructure is properly secured
- Consider network-level access controls if metrics contain sensitive
  information
- Metrics do not expose user data or credentials by design

## Troubleshooting

### Metrics Not Appearing

1. Verify the `/metrics` endpoint is accessible:
   `curl http://localhost:8080/metrics`
2. Check that the application is running with the correct base URL configuration
3. Ensure Prometheus is configured with the correct `metrics_path`

### Missing Provider Metrics

Provider metrics are only generated when subtitle requests are made. Test with a
subtitle download to generate metrics.

### High Memory Usage

If you're experiencing high memory usage with metrics enabled, consider:

- Reducing the number of label combinations
- Implementing metric retention policies
- Using external metrics storage for long-term retention

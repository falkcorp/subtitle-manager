### Security

#### Bump OpenTelemetry OTLP HTTP exporter to 1.43.0

Upgraded `go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp`
1.37.0 → 1.43.0 (and the OTLP trace exporter / proto packages it versions with)
to clear the Dependabot advisory. `github.com/docker/docker` stays at
`v28.5.2+incompatible` — it is already the latest version published to the Go
module proxy (there is no `v29.x`), so its remaining advisories have no upgrade
target yet.

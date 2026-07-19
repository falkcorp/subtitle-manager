// file: pkg/gcommonlog/logrus_provider.go
// version: 2.0.0
// guid: a1e4bca3-27a2-4e4a-9d4c-1c2f3b4d5e6f

package gcommonlog

import (
	"context"
	"io"

	common "github.com/falkcorp/gcommon/v2/pkg/commonpb/v2"
	"github.com/sirupsen/logrus"
)

type LogrusProvider struct {
	entry *logrus.Entry
}

// NewLogrusProvider creates a logrus-based logging provider compatible with gcommon protobuf types.
func NewLogrusProvider(cfg *common.LoggerConfig) (*LogrusProvider, error) {
	l := logrus.New()
	lvl := convertLogLevel(cfg.GetLevel())
	l.SetLevel(lvl)
	return &LogrusProvider{entry: logrus.NewEntry(l)}, nil
}

// NewLogrusProviderWithLevel creates a logrus provider with a specific log level.
func NewLogrusProviderWithLevel(level common.LogLevel) *LogrusProvider {
	l := logrus.New()
	lvl := convertLogLevel(level)
	l.SetLevel(lvl)
	return &LogrusProvider{entry: logrus.NewEntry(l)}
}

func (p *LogrusProvider) Name() string          { return "logrus" }
func (p *LogrusProvider) Close() error          { return nil }
func (p *LogrusProvider) Sync() error           { return nil }
func (p *LogrusProvider) SetOutput(w io.Writer) { p.entry.Logger.SetOutput(w) }

func (p *LogrusProvider) log(level logrus.Level, msg string, fields map[string]interface{}) {
	if !p.entry.Logger.IsLevelEnabled(level) {
		return
	}
	p.entry.WithFields(fields).Log(level, msg)
}

func (p *LogrusProvider) Debug(msg string, fields map[string]interface{}) {
	p.log(logrus.DebugLevel, msg, fields)
}

func (p *LogrusProvider) DebugContext(ctx context.Context, msg string, fields map[string]interface{}) {
	p.log(logrus.DebugLevel, msg, fields)
}

func (p *LogrusProvider) Info(msg string, fields map[string]interface{}) {
	p.log(logrus.InfoLevel, msg, fields)
}

func (p *LogrusProvider) InfoContext(ctx context.Context, msg string, fields map[string]interface{}) {
	p.log(logrus.InfoLevel, msg, fields)
}

func (p *LogrusProvider) Warn(msg string, fields map[string]interface{}) {
	p.log(logrus.WarnLevel, msg, fields)
}

func (p *LogrusProvider) WarnContext(ctx context.Context, msg string, fields map[string]interface{}) {
	p.log(logrus.WarnLevel, msg, fields)
}

func (p *LogrusProvider) Error(msg string, fields map[string]interface{}) {
	p.log(logrus.ErrorLevel, msg, fields)
}

func (p *LogrusProvider) ErrorContext(ctx context.Context, msg string, fields map[string]interface{}) {
	p.log(logrus.ErrorLevel, msg, fields)
}

func (p *LogrusProvider) Fatal(msg string, fields map[string]interface{}) {
	p.log(logrus.FatalLevel, msg, fields)
}

func (p *LogrusProvider) FatalContext(ctx context.Context, msg string, fields map[string]interface{}) {
	p.log(logrus.FatalLevel, msg, fields)
}

func (p *LogrusProvider) With(fields map[string]interface{}) *LogrusProvider {
	return &LogrusProvider{entry: p.entry.WithFields(fields)}
}

func (p *LogrusProvider) WithContext(ctx context.Context) *LogrusProvider {
	return &LogrusProvider{entry: p.entry.WithContext(ctx)}
}

func (p *LogrusProvider) SetLevel(l common.LogLevel) {
	p.entry.Logger.SetLevel(convertLogLevel(l))
}

func (p *LogrusProvider) GetLevel() common.LogLevel {
	return convertFromLogrusLevel(p.entry.Logger.GetLevel())
}

// convertLogLevel converts gcommon protobuf LogLevel to logrus Level
func convertLogLevel(l common.LogLevel) logrus.Level {
	switch l {
	case common.LogLevel_LOG_LEVEL_TRACE:
		return logrus.TraceLevel
	case common.LogLevel_LOG_LEVEL_DEBUG:
		return logrus.DebugLevel
	case common.LogLevel_LOG_LEVEL_INFO:
		return logrus.InfoLevel
	case common.LogLevel_LOG_LEVEL_WARN:
		return logrus.WarnLevel
	case common.LogLevel_LOG_LEVEL_ERROR:
		return logrus.ErrorLevel
	case common.LogLevel_LOG_LEVEL_FATAL:
		return logrus.FatalLevel
	default:
		return logrus.InfoLevel
	}
}

// convertFromLogrusLevel converts logrus Level to gcommon protobuf LogLevel
func convertFromLogrusLevel(l logrus.Level) common.LogLevel {
	switch l {
	case logrus.TraceLevel:
		return common.LogLevel_LOG_LEVEL_TRACE
	case logrus.DebugLevel:
		return common.LogLevel_LOG_LEVEL_DEBUG
	case logrus.InfoLevel:
		return common.LogLevel_LOG_LEVEL_INFO
	case logrus.WarnLevel:
		return common.LogLevel_LOG_LEVEL_WARN
	case logrus.ErrorLevel:
		return common.LogLevel_LOG_LEVEL_ERROR
	case logrus.FatalLevel:
		return common.LogLevel_LOG_LEVEL_FATAL
	default:
		return common.LogLevel_LOG_LEVEL_INFO
	}
}

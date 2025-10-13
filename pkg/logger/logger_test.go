package logger

import (
	"context"
	"testing"
)

func TestNew(t *testing.T) {
	tests := []struct {
		name    string
		level   string
		format  string
		wantErr bool
	}{
		{
			name:    "create logger with info level and json format",
			level:   "info",
			format:  "json",
			wantErr: false,
		},
		{
			name:    "create logger with debug level and console format",
			level:   "debug",
			format:  "console",
			wantErr: false,
		},
		{
			name:    "create logger with invalid level (should default to info)",
			level:   "invalid",
			format:  "json",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger, err := New(tt.level, tt.format)
			if (err != nil) != tt.wantErr {
				t.Errorf("New() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if logger == nil && !tt.wantErr {
				t.Error("New() returned nil logger")
			}
		})
	}
}

func TestNewDefault(t *testing.T) {
	logger := NewDefault()
	if logger == nil {
		t.Error("NewDefault() returned nil")
	}

	// 기본 로깅 테스트
	logger.Info("test message", "key", "value")
	logger.Debug("debug message")
	logger.Warn("warning message")
	logger.Error("error message")
}

func TestZapLogger_WithContext(t *testing.T) {
	logger := NewDefault()

	ctx := context.Background()
	ctx = context.WithValue(ctx, "trace_id", "test-trace-123")
	ctx = context.WithValue(ctx, "request_id", "test-request-456")

	contextLogger := logger.WithContext(ctx)
	if contextLogger == nil {
		t.Error("WithContext() returned nil")
	}

	contextLogger.Info("test with context")
}

func TestZapLogger_WithFields(t *testing.T) {
	logger := NewDefault()

	fields := map[string]interface{}{
		"user_id":    "12345",
		"request_id": "req-789",
		"method":     "GET",
	}

	fieldLogger := logger.WithFields(fields)
	if fieldLogger == nil {
		t.Error("WithFields() returned nil")
	}

	fieldLogger.Info("test with fields")
}

func TestSimpleLogger(t *testing.T) {
	logger := NewSimpleLogger("info")
	if logger == nil {
		t.Error("NewSimpleLogger() returned nil")
	}

	logger.Debug("debug message")
	logger.Info("info message", "key", "value")
	logger.Warn("warning message")
	logger.Error("error message")

	ctx := context.Background()
	contextLogger := logger.WithContext(ctx)
	contextLogger.Info("context message")

	fields := map[string]interface{}{"field": "value"}
	fieldLogger := logger.WithFields(fields)
	fieldLogger.Info("field message")
}

func BenchmarkLogger_Info(b *testing.B) {
	logger := NewDefault()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.Info("benchmark message", "iteration", i)
	}
}

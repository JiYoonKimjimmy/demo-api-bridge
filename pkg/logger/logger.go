package logger

import (
	"context"
	"demo-api-bridge/internal/core/port"
	"fmt"
	"os"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// zapLogger는 Zap 기반 Logger 구현체입니다.
type zapLogger struct {
	logger *zap.Logger
	sugar  *zap.SugaredLogger
}

// New는 새로운 Logger를 생성합니다.
func New(level string, format string) (port.Logger, error) {
	config := zap.NewProductionConfig()

	// 로그 레벨 설정
	logLevel, err := zapcore.ParseLevel(level)
	if err != nil {
		logLevel = zapcore.InfoLevel
	}
	config.Level = zap.NewAtomicLevelAt(logLevel)

	// 로그 포맷 설정
	if format == "console" {
		config.Encoding = "console"
		config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
		config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	} else {
		config.Encoding = "json"
		config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	}

	// 출력 설정
	config.OutputPaths = []string{"stdout"}
	config.ErrorOutputPaths = []string{"stderr"}

	// Logger 생성
	logger, err := config.Build(
		zap.AddCallerSkip(1), // caller 정보를 위해 skip
		zap.AddStacktrace(zapcore.ErrorLevel),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create logger: %w", err)
	}

	return &zapLogger{
		logger: logger,
		sugar:  logger.Sugar(),
	}, nil
}

// NewDefault는 기본 설정의 Logger를 생성합니다.
func NewDefault() port.Logger {
	logger, err := New("info", "console")
	if err != nil {
		// Fallback to basic logger
		return &zapLogger{
			logger: zap.NewExample(),
			sugar:  zap.NewExample().Sugar(),
		}
	}
	return logger
}

// Debug는 디버그 레벨 로그를 출력합니다.
func (l *zapLogger) Debug(msg string, fields ...interface{}) {
	l.sugar.Debugw(msg, fields...)
}

// Info는 정보 레벨 로그를 출력합니다.
func (l *zapLogger) Info(msg string, fields ...interface{}) {
	l.sugar.Infow(msg, fields...)
}

// Warn은 경고 레벨 로그를 출력합니다.
func (l *zapLogger) Warn(msg string, fields ...interface{}) {
	l.sugar.Warnw(msg, fields...)
}

// Error는 에러 레벨 로그를 출력합니다.
func (l *zapLogger) Error(msg string, fields ...interface{}) {
	l.sugar.Errorw(msg, fields...)
}

// WithContext는 컨텍스트를 포함한 로거를 반환합니다.
func (l *zapLogger) WithContext(ctx context.Context) port.Logger {
	// 컨텍스트에서 trace_id 등을 추출하여 필드에 추가
	fields := make(map[string]interface{})

	if traceID := ctx.Value("trace_id"); traceID != nil {
		fields["trace_id"] = traceID
	}

	if requestID := ctx.Value("request_id"); requestID != nil {
		fields["request_id"] = requestID
	}

	if len(fields) > 0 {
		return l.WithFields(fields)
	}

	return l
}

// WithFields는 필드를 포함한 로거를 반환합니다.
func (l *zapLogger) WithFields(fields map[string]interface{}) port.Logger {
	zapFields := make([]zap.Field, 0, len(fields))
	for k, v := range fields {
		zapFields = append(zapFields, zap.Any(k, v))
	}

	return &zapLogger{
		logger: l.logger.With(zapFields...),
		sugar:  l.logger.With(zapFields...).Sugar(),
	}
}

// Sync는 버퍼를 플러시합니다.
func (l *zapLogger) Sync() error {
	return l.logger.Sync()
}

// SimpleLogger는 간단한 로거 구현체입니다 (테스트용).
type SimpleLogger struct {
	level string
}

// NewSimpleLogger는 간단한 로거를 생성합니다.
func NewSimpleLogger(level string) port.Logger {
	return &SimpleLogger{level: level}
}

func (l *SimpleLogger) Debug(msg string, fields ...interface{}) {
	l.log("DEBUG", msg, fields...)
}

func (l *SimpleLogger) Info(msg string, fields ...interface{}) {
	l.log("INFO", msg, fields...)
}

func (l *SimpleLogger) Warn(msg string, fields ...interface{}) {
	l.log("WARN", msg, fields...)
}

func (l *SimpleLogger) Error(msg string, fields ...interface{}) {
	l.log("ERROR", msg, fields...)
}

func (l *SimpleLogger) WithContext(ctx context.Context) port.Logger {
	return l
}

func (l *SimpleLogger) WithFields(fields map[string]interface{}) port.Logger {
	return l
}

func (l *SimpleLogger) log(level, msg string, fields ...interface{}) {
	timestamp := time.Now().Format(time.RFC3339)
	fmt.Fprintf(os.Stdout, "[%s] %s: %s", timestamp, level, msg)
	if len(fields) > 0 {
		fmt.Fprintf(os.Stdout, " %v", fields)
	}
	fmt.Fprintln(os.Stdout)
}

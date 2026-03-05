package tools

import (
	"context"
	"regexp"
	"time"

	"gorm.io/gorm/logger"
)

// redactLogger wraps a GORM logger and redacts sensitive values in SQL before logging.
type redactLogger struct {
	underlying logger.Interface
}

// RedactLogger returns a logger that redacts password hashes, tokens, and emails in SQL logs.
func RedactLogger(l logger.Interface) logger.Interface {
	return &redactLogger{underlying: l}
}

func (r *redactLogger) LogMode(level logger.LogLevel) logger.Interface {
	return &redactLogger{underlying: r.underlying.LogMode(level)}
}

func (r *redactLogger) Info(ctx context.Context, msg string, args ...interface{}) {
	r.underlying.Info(ctx, msg, args...)
}

func (r *redactLogger) Warn(ctx context.Context, msg string, args ...interface{}) {
	r.underlying.Warn(ctx, msg, args...)
}

func (r *redactLogger) Error(ctx context.Context, msg string, args ...interface{}) {
	r.underlying.Error(ctx, msg, args...)
}

func (r *redactLogger) Trace(ctx context.Context, begin time.Time, fc func() (sql string, rowsAffected int64), err error) {
	sql, rows := fc()
	redacted := redactSensitiveSQL(sql)
	r.underlying.Trace(ctx, begin, func() (string, int64) { return redacted, rows }, err)
}

// redactSensitiveSQL replaces sensitive values in SQL with [REDACTED].
// - Single-quoted strings of 40+ chars (e.g. password hashes, tokens)
// - Single-quoted strings containing @ (emails)
var (
	redactLongString = regexp.MustCompile(`'[^']{40,}'`)
	redactEmail      = regexp.MustCompile(`'[^']*@[^']*'`)
)

func redactSensitiveSQL(sql string) string {
	// Redact email-like values first
	s := redactEmail.ReplaceAllString(sql, "'[REDACTED]'")
	// Redact long strings (hashes, tokens)
	s = redactLongString.ReplaceAllString(s, "'[REDACTED]'")
	return s
}

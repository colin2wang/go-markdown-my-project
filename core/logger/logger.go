// Package logger 提供基于 log/slog 的日志初始化。
// 对齐旧版 log4rs.yml：控制台 + 滚动文件双输出，
// pattern: {time} {level} {file}:{line} — {msg}
package logger

import (
	"context"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/natefinch/lumberjack.v2"
)

var global *slog.Logger = slog.New(slog.NewTextHandler(io.Discard, nil))

// Options 日志初始化选项，对齐 log4rs.yml。
type Options struct {
	Level      string // debug | info | warn | error
	LogDir     string // 日志目录，如 "logs"
	LogFile    string // 文件名，如 "project_documentation.log"
	MaxSizeMB  int    // 单文件上限，超过滚动（默认 1MB）
	MaxBackups int    // 保留的历史文件数（默认 10）
	Console    bool   // 是否同时输出到控制台
}

// Init 初始化全局 slog logger。默认 debug 级别、控制台 + 滚动文件双输出。
func Init(opts Options) error {
	if opts.Level == "" {
		opts.Level = "debug"
	}
	var level slog.Level
	switch opts.Level {
	case "debug":
		level = slog.LevelDebug
	case "info":
		level = slog.LevelInfo
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		level = slog.LevelDebug
	}

	if opts.MaxSizeMB <= 0 {
		opts.MaxSizeMB = 1
	}
	if opts.MaxBackups <= 0 {
		opts.MaxBackups = 10
	}
	if opts.LogFile == "" {
		opts.LogFile = "project_documentation.log"
	}
	if opts.LogDir == "" {
		opts.LogDir = "logs"
	}

	var writers []io.Writer
	if opts.Console {
		writers = append(writers, os.Stdout)
	}
	if err := os.MkdirAll(opts.LogDir, 0o755); err != nil {
		return err
	}
	writers = append(writers, &lumberjack.Logger{
		Filename:   filepath.Join(opts.LogDir, opts.LogFile),
		MaxSize:    opts.MaxSizeMB,
		MaxBackups: opts.MaxBackups,
		Compress:   false,
	})

	handler := slog.NewTextHandler(io.MultiWriter(writers...), &slog.HandlerOptions{
		Level: level,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			// 对齐 pattern：时间格式 2006-01-02 15:04:05
			if a.Key == slog.TimeKey {
				if t, ok := a.Value.Any().(time.Time); ok {
					a.Value = slog.StringValue(t.Format("2006-01-02 15:04:05"))
				}
			}
			// 对齐 {f} {L}：仅源文件名 + 行号
			if a.Key == slog.SourceKey {
				if s, ok := a.Value.Any().(*slog.Source); ok {
					a.Value = slog.StringValue(filepath.Base(s.File) + ":" + itoa(s.Line))
				}
			}
			return a
		},
		AddSource: true,
	})
	global = slog.New(handler)
	slog.SetDefault(global)
	return nil
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b [20]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	return string(b[i:])
}

// L 返回全局 logger。
func L() *slog.Logger { return global }

// 便捷方法
func Debug(msg string, args ...any) { global.Log(context.Background(), slog.LevelDebug, msg, args...) }
func Info(msg string, args ...any)  { global.Log(context.Background(), slog.LevelInfo, msg, args...) }
func Warn(msg string, args ...any)  { global.Log(context.Background(), slog.LevelWarn, msg, args...) }
func Error(msg string, args ...any) { global.Log(context.Background(), slog.LevelError, msg, args...) }

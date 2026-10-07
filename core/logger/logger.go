// Package logger 提供基于 log/slog 的日志初始化。
// 对齐旧版 log4rs.yml：控制台 + 滚动文件双输出，
// pattern: {time} {level} {file}:{line} — {msg}
package logger

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/natefinch/lumberjack.v2"
)

var global *slog.Logger = slog.New(slog.NewTextHandler(io.Discard, nil))

// Entry 一条日志的结构化表示，用于转发到前端日志窗口。
type Entry struct {
	Time   string `json:"time"`
	Level  string `json:"level"`
	Msg    string `json:"msg"`
	Source string `json:"source"`
}

// sink 日志转发回调；由 app 层注入（core 不依赖 Wails）。
var sink func(Entry)

// SetSink 设置日志转发回调（传 nil 关闭转发）。
func SetSink(fn func(Entry)) { sink = fn }

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
	// 目录创建失败不应阻断整个日志系统：降级为仅控制台输出，
	// 并始终挂上 sinkHandler，保证前端日志转发与后续文件日志可用。
	if err := os.MkdirAll(opts.LogDir, 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "[logger] 无法创建日志目录 %q: %v（仅控制台输出）\n", opts.LogDir, err)
	} else {
		writers = append(writers, &lumberjack.Logger{
			Filename:   filepath.Join(opts.LogDir, opts.LogFile),
			MaxSize:    opts.MaxSizeMB,
			MaxBackups: opts.MaxBackups,
			Compress:   false,
		})
	}

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
	// 总是包裹 sinkHandler：sink 可能在 Init 之后（如 Startup）才注入，
	// 转发在 Handle 时按 sink 是否为 nil 动态决定，避免错过后续日志。
	global = slog.New(&sinkHandler{base: handler})
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

// sinkHandler 包裹底层 handler，在每条日志记录落盘/控制台的同时转发给 sink（前端日志窗口）。
type sinkHandler struct{ base slog.Handler }

func (h *sinkHandler) Enabled(ctx context.Context, l slog.Level) bool {
	return h.base.Enabled(ctx, l)
}

func (h *sinkHandler) Handle(ctx context.Context, r slog.Record) error {
	err := h.base.Handle(ctx, r)
	if sink != nil {
		e := Entry{
			Time:  r.Time.Format("2006-01-02 15:04:05"),
			Level: r.Level.String(),
			Msg:   r.Message,
		}
		if src := r.Source(); src != nil {
			e.Source = filepath.Base(src.File) + ":" + itoa(src.Line)
		}
		sink(e)
	}
	return err
}

func (h *sinkHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &sinkHandler{base: h.base.WithAttrs(attrs)}
}

func (h *sinkHandler) WithGroup(name string) slog.Handler {
	return &sinkHandler{base: h.base.WithGroup(name)}
}

// 便捷方法
func Debug(msg string, args ...any) { global.Log(context.Background(), slog.LevelDebug, msg, args...) }
func Info(msg string, args ...any)  { global.Log(context.Background(), slog.LevelInfo, msg, args...) }
func Warn(msg string, args ...any)  { global.Log(context.Background(), slog.LevelWarn, msg, args...) }
func Error(msg string, args ...any) { global.Log(context.Background(), slog.LevelError, msg, args...) }

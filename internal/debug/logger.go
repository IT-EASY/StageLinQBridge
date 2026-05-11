package debug

import (
	"log"
	"os"
)

type Logger interface {
	Level() Level

	Error(msg string, args ...any)
	Warn(msg string, args ...any)
	Info(msg string, args ...any)
	Debug(msg string, args ...any)
	Trace(msg string, args ...any)
}

type StandardLogger struct {
	level  Level
	logger *log.Logger
}

func New(level Level) *StandardLogger {
	return &StandardLogger{
		level: level,
		logger: log.New(
			os.Stdout,
			"",
			log.LstdFlags|log.Lmicroseconds,
		),
	}
}

func (l *StandardLogger) Level() Level {
	return l.level
}

func (l *StandardLogger) log(level Level, prefix string, msg string, args ...any) {
	if l.level < level {
		return
	}

	values := append([]any{prefix, msg}, args...)

	l.logger.Println(values...)
}

func (l *StandardLogger) Error(msg string, args ...any) {
	l.log(Error, "[ERROR]", msg, args...)
}

func (l *StandardLogger) Warn(msg string, args ...any) {
	l.log(Warn, "[WARN]", msg, args...)
}

func (l *StandardLogger) Info(msg string, args ...any) {
	l.log(Info, "[INFO]", msg, args...)
}

func (l *StandardLogger) Debug(msg string, args ...any) {
	l.log(Debug, "[DEBUG]", msg, args...)
}

func (l *StandardLogger) Trace(msg string, args ...any) {
	l.log(Trace, "[TRACE]", msg, args...)
}

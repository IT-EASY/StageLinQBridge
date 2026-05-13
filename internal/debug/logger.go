package debug

import (
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"time"
)

type Level int

const (
	Error Level = iota
	Warn
	Info
	Debug
	Trace
)

func (l Level) String() string {
	switch l {
	case Error:
		return "ERROR"
	case Warn:
		return "WARN"
	case Info:
		return "INFO"
	case Debug:
		return "DEBUG"
	case Trace:
		return "TRACE"
	default:
		return "UNKNOWN"
	}
}

type Logger struct {
	level  Level
	logger *log.Logger
	mutex  sync.Mutex
}

func New(level Level) *Logger {
	return &Logger{
		level: level,
		logger: log.New(
			os.Stdout,
			"",
			0,
		),
	}
}

func (l *Logger) Error(message string, args ...any) {
	l.log(Error, message, args...)
}

func (l *Logger) Warn(message string, args ...any) {
	l.log(Warn, message, args...)
}

func (l *Logger) Info(message string, args ...any) {
	l.log(Info, message, args...)
}

func (l *Logger) Debug(message string, args ...any) {
	l.log(Debug, message, args...)
}

func (l *Logger) Trace(message string, args ...any) {
	l.log(Trace, message, args...)
}

func (l *Logger) log(level Level, message string, args ...any) {
	if level > l.level {
		return
	}

	l.mutex.Lock()
	defer l.mutex.Unlock()

	timestamp := time.Now().Format("2006-01-02 15:04:05")

	var builder strings.Builder

	builder.WriteString(timestamp)
	builder.WriteString(" [")
	builder.WriteString(level.String())
	builder.WriteString("] ")
	builder.WriteString(message)

	for i := 0; i < len(args)-1; i += 2 {
		key := fmt.Sprint(args[i])
		value := fmt.Sprint(args[i+1])

		builder.WriteString(" ")
		builder.WriteString(key)
		builder.WriteString("=")
		builder.WriteString(value)
	}

	l.logger.Println(builder.String())
}

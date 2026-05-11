package debug

type NoopLogger struct{}

func NewNoop() *NoopLogger {
	return &NoopLogger{}
}

func (l *NoopLogger) Level() Level {
	return Off
}

func (l *NoopLogger) Error(string, ...any) {}
func (l *NoopLogger) Warn(string, ...any)  {}
func (l *NoopLogger) Info(string, ...any)  {}
func (l *NoopLogger) Debug(string, ...any) {}
func (l *NoopLogger) Trace(string, ...any) {}

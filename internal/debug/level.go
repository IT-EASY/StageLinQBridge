package debug

type Level int

const (
	Off Level = iota
	Error
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
		return "OFF"
	}
}

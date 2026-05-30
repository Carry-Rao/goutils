package log

type LogLevel int

const (
	Debug LogLevel = 1
	Info  LogLevel = 2
	Warn  LogLevel = 3
	Error LogLevel = 4
	Panic LogLevel = 5
)

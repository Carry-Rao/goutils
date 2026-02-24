package log

import (
	"os"
)

type Level int

type Logger interface {
	Debug(format string, p ...any)
	Info(format string, p ...any)
	Warn(format string, p ...any)
	Error(format string, p ...any)
	Fatal(format string, p ...any)
	Panic(format string, p ...any)
	SetLevel(level Level)
}

func NewFileLogger(path string) Logger {
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		panic(err)
	}
	return NewFile(file)
}

func NewConsoleLogger(stdout bool) Logger {
	return NewConsole(stdout)
}

const (
	Debug Level = 0
	Info  Level = 1
	Warn  Level = 2
	Error Level = 3
	Fatal Level = 4
	Panic Level = 5
)

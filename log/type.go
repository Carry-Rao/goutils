package log

import (
	"os"
)

type Level int

type Logger interface {
	Debug(p ...any)
	Info(p ...any)
	Warn(p ...any)
	Error(p ...any)
	Fatal(p ...any)
	Panic(p ...any)
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

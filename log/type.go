package log

import (
	"os"
)

type Logger interface {
	Debug(p ...any)
	Info(p ...any)
	Warn(p ...any)
	Error(p ...any)
	Fatal(p ...any)
	Panic(p ...any)
	SetLevel(level int)
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
	Debug = 0
	Info  = 1
	Warn  = 2
	Error = 3
	Fatal = 4
	Panic = 5
)

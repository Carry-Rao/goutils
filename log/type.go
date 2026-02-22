package log

import (
	"io"
	"os"
)

type Logger interface {
	Debug(p string)
	Info(p string)
	Warn(p string)
	Error(p string)
	Fatal(p string)
	Panic(p string)
	SetLevel(level int)
}

func NewFileLogger(path string) Logger {
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		panic(err)
	}
	writer := io.Writer(file)
	return NewFile(&writer)
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

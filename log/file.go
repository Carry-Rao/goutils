package log

import (
	"fmt"
	"os"
	"time"
)

type File struct {
	Level Level
	file  *os.File
}

func NewFile(file *os.File) *File {
	return &File{file: file, Level: 1}
}

func (f *File) Debug(format string, p ...any) {
	if f.Level > 0 {
		return
	}
	fmt.Fprintf(f.file, "[Debug] %s %s\n", time.Now().Format("2006-01-02 15:04:05"), " ")
	fmt.Fprintf(f.file, format, p...)
	fmt.Fprintln(f.file)
}

func (f *File) Info(format string, p ...any) {
	if f.Level > 1 {
		return
	}
	fmt.Fprintf(f.file, "[Info] %s %s\n", time.Now().Format("2006-01-02 15:04:05"), " ")
	fmt.Fprintf(f.file, format, p...)
	fmt.Fprintln(f.file)
}

func (f *File) Warn(format string, p ...any) {
	if f.Level > 2 {
		return
	}
	fmt.Fprintf(f.file, "[Warn] %s %s\n", time.Now().Format("2006-01-02 15:04:05"), " ")
	fmt.Fprintf(f.file, format, p...)
	fmt.Fprintln(f.file)
}

func (f *File) Error(format string, p ...any) {
	if f.Level > 3 {
		return
	}
	fmt.Fprintf(f.file, "[Error] %s %s\n", time.Now().Format("2006-01-02 15:04:05"), " ")
	fmt.Fprintf(f.file, format, p...)
	fmt.Fprintln(f.file)
}

func (f *File) Fatal(format string, p ...any) {
	if f.Level > 4 {
		return
	}
	fmt.Fprintf(f.file, "[Fatal] %s %s\n", time.Now().Format("2006-01-02 15:04:05"), " ")
	fmt.Fprintf(f.file, format, p...)
	fmt.Fprintln(f.file)
	os.Exit(1)
}

func (f *File) Panic(format string, p ...any) {
	if f.Level > 5 {
		return
	}
	fmt.Fprintf(f.file, "[Panic] %s %s\n", time.Now().Format("2006-01-02 15:04:05"), " ")
	fmt.Fprintf(f.file, format, p...)
	fmt.Fprintln(f.file)
	panic(fmt.Sprintf(format, p...))
}

func (f *File) SetLevel(level Level) {
	f.Level = level
}

package log

import (
	"fmt"
	"os"
	"time"
)

type File struct {
	Level int
	file  *os.File
}

func NewFile(file *os.File) *File {
	return &File{file: file, Level: 1}
}

func (f *File) Debug(p ...any) {
	if f.Level > 0 {
		return
	}
	fmt.Fprintf(f.file, "[Debug] %s %s\n", time.Now().Format("2006-01-02 15:04:05"), " ")
	fmt.Fprintln(f.file, p)
}

func (f *File) Info(p ...any) {
	if f.Level > 1 {
		return
	}
	fmt.Fprintf(f.file, "[Info] %s %s\n", time.Now().Format("2006-01-02 15:04:05"), " ")
	fmt.Fprintln(f.file, p)
}

func (f *File) Warn(p ...any) {
	if f.Level > 2 {
		return
	}
	fmt.Fprintf(f.file, "[Warn] %s %s\n", time.Now().Format("2006-01-02 15:04:05"), " ")
	fmt.Fprintln(f.file, p)
}

func (f *File) Error(p ...any) {
	if f.Level > 3 {
		return
	}
	fmt.Fprintf(f.file, "[Error] %s %s\n", time.Now().Format("2006-01-02 15:04:05"), " ")
	fmt.Fprintln(f.file, p)
}

func (f *File) Fatal(p ...any) {
	if f.Level > 4 {
		return
	}
	fmt.Fprintf(f.file, "[Fatal] %s %s\n", time.Now().Format("2006-01-02 15:04:05"), " ")
	fmt.Fprintln(f.file, p)
	os.Exit(1)
}

func (f *File) Panic(p ...any) {
	if f.Level > 5 {
		return
	}
	fmt.Fprintf(f.file, "[Panic] %s %s\n", time.Now().Format("2006-01-02 15:04:05"), " ")
	fmt.Fprintln(f.file, p)
	panic(p)
}

func (f *File) SetLevel(level int) {
	f.Level = level
}

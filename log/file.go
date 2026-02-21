package log

import (
	"io"
	"os"
	"time"
)

type File struct {
	Level  int
	Writer *io.Writer
}

func NewFile(writer *io.Writer) *File {
	return &File{Writer: writer, Level: 1}
}

func (f *File) Debug(p string) {
	if f.Level > 0 {
		return
	}
	(*f.Writer).Write([]byte("[Debug] " + time.Now().Format("2006-01-02 15:04:05") + " " + p + "\n"))
}

func (f *File) Info(p string) {
	if f.Level > 1 {
		return
	}
	(*f.Writer).Write([]byte("[Info] " + time.Now().Format("2006-01-02 15:04:05") + " " + p + "\n"))
}

func (f *File) Warn(p string) {
	if f.Level > 2 {
		return
	}
	(*f.Writer).Write([]byte("[Warn] " + time.Now().Format("2006-01-02 15:04:05") + " " + p + "\n"))
}

func (f *File) Error(p string) {
	if f.Level > 3 {
		return
	}
	(*f.Writer).Write([]byte("[Error] " + time.Now().Format("2006-01-02 15:04:05") + " " + p + "\n"))
}

func (f *File) Fatal(p string) {
	if f.Level > 4 {
		return
	}
	(*f.Writer).Write([]byte("[Fatal] " + time.Now().Format("2006-01-02 15:04:05") + " " + p + "\n"))
	os.Exit(1)
}

func (f *File) Panic(p string) {
	if f.Level > 5 {
		return
	}
	(*f.Writer).Write([]byte("[Panic] " + time.Now().Format("2006-01-02 15:04:05") + " " + p + "\n"))
	panic(p)
}

func (f *File) SetLevel(level int) {
	f.Level = level
}

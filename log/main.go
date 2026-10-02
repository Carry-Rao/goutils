package log

import (
	"os"
	"time"
)

const (
	timeFmt   = "2006-01-02 15:04:05"
	nlLen     = 1
	ansiGray  = "\033[90m"
	ansiReset = "\033[0m"
	space     = " "
)

// entryLen must stay in step with writeEntry, or entries corrupt each other.
func (l *Logger) entryLen(ts string, msg []byte) uint32 {
	n := len(ts) + len(space) + len(msg) + nlLen
	if l.Color {
		n += len(ansiGray) + len(ansiReset)
	}
	return uint32(n)
}

// writeEntry appends one entry to dst. A zero-length dst over spare buffer
// capacity is filled in place.
func (l *Logger) writeEntry(dst []byte, ts string, msg []byte) []byte {
	if l.Color {
		dst = append(dst, ansiGray...)
	}
	dst = append(dst, ts...)
	if l.Color {
		dst = append(dst, ansiReset...)
	}
	dst = append(dst, space...)
	dst = append(dst, msg...)
	return append(dst, '\n')
}

// writeDirect bypasses the buffer, for entries too large to buffer.
func (l *Logger) writeDirect(dst []byte, ts string, msg []byte) {
	l.mu.Lock()
	defer l.mu.Unlock()

	_, _ = l.File.Write(l.writeEntry(dst, ts, msg))
}

// appendBuffered adds one entry, flushing first when full. The lock covers the
// whole append: claiming space and then writing it separately let a concurrent
// Flush emit a not-yet-filled region twice.
func (l *Logger) appendBuffered(ts string, msg []byte, size uint32) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.Pos+size > l.Len {
		l.flushLocked()
	}

	off := l.Pos
	l.writeEntry(l.Buffer[off:off], ts, msg)
	l.Pos += size
}

func (l *Logger) Flush() {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.flushLocked()
}

func (l *Logger) flushLocked() {
	if l.Pos == 0 {
		return
	}
	n := l.Pos
	l.Pos = 0
	_, _ = l.File.Write(l.Buffer[:n])
}

func (l *Logger) addLog(log Log) {
	ts := time.Now().Format(timeFmt)
	msg := []byte(log.LogInfo)
	size := l.entryLen(ts, msg)

	// Too large to ever buffer.
	if size > l.Len {
		l.writeDirect(make([]byte, 0, size), ts, msg)
		return
	}
	l.appendBuffered(ts, msg, size)
}

// NewLogger opens file for appending. It panics if the file cannot be opened.
func NewLogger(file string, bufferLen uint32) *Logger {
	f, err := os.OpenFile(file, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		panic(err)
	}
	return &Logger{
		File:     f,
		LogLevel: Info,
		Buffer:   make([]byte, bufferLen),
		Len:      bufferLen,
		Color:    false,
	}
}

var Console = &Logger{
	File:     os.Stdout,
	LogLevel: Info,
	Buffer:   make([]byte, 4096),
	Len:      4096,
	Color:    true,
}

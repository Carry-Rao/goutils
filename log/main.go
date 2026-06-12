package log

import (
	"os"
	"sync/atomic"
	"time"
)

const (
	timeFmt   = "2006-01-02 15:04:05"
	nlLen     = 1
	ansiGray  = "\033[90m"
	ansiReset = "\033[0m"
	space     = " "
)

func (l *Logger) addLog(log Log) {
	ts := time.Now().Format(timeFmt)
	logB := []byte(log.LogInfo)

	var entryLen uint32
	if l.Color {
		entryLen = uint32(len(ansiGray) + len(ts) + len(ansiReset) + len(space) + len(logB) + nlLen)
	} else {
		entryLen = uint32(len(ts) + len(space) + len(logB) + nlLen)
	}

	if entryLen > l.Len {
		var buf []byte
		if l.Color {
			buf = append(buf, ansiGray...)
			buf = append(buf, ts...)
			buf = append(buf, ansiReset...)
		} else {
			buf = append(buf, ts...)
		}
		buf = append(buf, space...)
		buf = append(buf, logB...)
		buf = append(buf, '\n')
		_, _ = l.File.Write(buf)
		return
	}

	for {
		oldPos := atomic.LoadUint32(&l.Pos)
		if oldPos+entryLen > l.Len {
			l.Flush()
			atomic.CompareAndSwapUint32(&l.Pos, oldPos, 0)
			continue
		}
		newPos := oldPos + entryLen
		if !atomic.CompareAndSwapUint32(&l.Pos, oldPos, newPos) {
			continue
		}

		off := oldPos
		if l.Color {
			copy(l.Buffer[off:], ansiGray)
			off += uint32(len(ansiGray))
		}
		copy(l.Buffer[off:], ts)
		off += uint32(len(ts))
		if l.Color {
			copy(l.Buffer[off:], ansiReset)
			off += uint32(len(ansiReset))
		}
		copy(l.Buffer[off:], space)
		off += uint32(len(space))
		copy(l.Buffer[off:], logB)
		off += uint32(len(logB))
		l.Buffer[off] = '\n'
		break
	}
}

func (l *Logger) Flush() {
	oldPos := atomic.SwapUint32(&l.Pos, 0)
	if oldPos == 0 {
		return
	}
	_, _ = l.File.Write(l.Buffer[:oldPos])
}

func NewLogger(file string, bufferLen uint32) Logger {
	f, err := os.OpenFile(file, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		panic(err)
	}
	return Logger{
		File:     f,
		LogLevel: Info,
		Buffer:   make([]byte, bufferLen),
		Len:      bufferLen,
		Color:    false,
	}
}

var Console = Logger{
	File:     os.Stdout,
	LogLevel: Info,
	Buffer:   make([]byte, 4096),
	Len:      4096,
	Color:    true,
}

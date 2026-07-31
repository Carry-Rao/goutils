package log

import (
	"io"
	"log"
	"os"
	"testing"
)

func newBenchLogger(b *testing.B) *Logger {
	b.Helper()
	f, err := os.CreateTemp("", "goutils-log-bench-*")
	if err != nil {
		b.Fatal(err)
	}
	b.Cleanup(func() {
		f.Close()
		os.Remove(f.Name())
	})
	return &Logger{
		File:     f,
		LogLevel: Debug,
		Buffer:   make([]byte, 4096),
		Len:      4096,
		Color:    false,
	}
}

func BenchmarkDebug(b *testing.B) {
	l := newBenchLogger(b)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		l.Debug("benchmark debug message")
	}
}

func BenchmarkInfo(b *testing.B) {
	l := newBenchLogger(b)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		l.Info("benchmark info message")
	}
}

func BenchmarkError(b *testing.B) {
	l := newBenchLogger(b)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		l.Error("benchmark error message")
	}
}

func BenchmarkInfoColor(b *testing.B) {
	l := newBenchLogger(b)
	l.Color = true
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		l.Info("benchmark info message")
	}
}

func BenchmarkInfoDiscard(b *testing.B) {
	l := newBenchLogger(b)
	l.File = os.NewFile(0, os.DevNull)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		l.Info("benchmark info message")
	}
}

func BenchmarkStdlibLog(b *testing.B) {
	f, err := os.CreateTemp("", "goutils-stdlib-log-*")
	if err != nil {
		b.Fatal(err)
	}
	b.Cleanup(func() {
		f.Close()
		os.Remove(f.Name())
	})
	l := log.New(f, "", log.LstdFlags)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		l.Print("benchmark info message")
	}
}

var _ = io.Discard

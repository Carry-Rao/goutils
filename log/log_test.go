package log

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

func newFileLogger(t *testing.T, bufLen uint32, color bool) (*Logger, string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "log.txt")
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { f.Close() })
	return &Logger{
		File:     f,
		LogLevel: Debug,
		Buffer:   make([]byte, bufLen),
		Len:      bufLen,
		Color:    color,
	}, path
}

// Entries must land in the file in order, with no gaps, overlaps or truncation.
func TestBufferedWriteOrdering(t *testing.T) {
	l, path := newFileLogger(t, 256, false)

	want := make([]string, 0, 50)
	for i := 0; i < 50; i++ {
		msg := "message-" + strings.Repeat("x", i%17) + "-" + itoa(i)
		l.Info(msg)
		want = append(want, "[INFO] "+msg)
	}
	l.Flush()

	got := readFile(t, path)
	lines := strings.Split(strings.TrimRight(got, "\n"), "\n")
	if len(lines) != len(want) {
		t.Fatalf("line count: got %d, want %d", len(lines), len(want))
	}
	for i := range want {
		if !strings.HasSuffix(lines[i], want[i]) {
			t.Errorf("line %d: got %q, want suffix %q", i, lines[i], want[i])
		}
	}
}

// A single entry larger than the buffer must bypass buffering entirely.
func TestOversizedEntryBypass(t *testing.T) {
	l, path := newFileLogger(t, 16, false)

	huge := strings.Repeat("z", 200)
	l.Info(huge)
	l.Info("after")
	l.Flush()

	got := readFile(t, path)
	if !strings.Contains(got, "[INFO] "+huge) {
		t.Error("oversized entry missing")
	}
	if !strings.Contains(got, "[INFO] after") {
		t.Error("entry after oversized one missing")
	}
}

// An oversized entry must not corrupt the entries buffered around it.
func TestOversizedEntryDoesNotCorruptNeighbours(t *testing.T) {
	l, path := newFileLogger(t, 128, false)

	for round := 0; round < 5; round++ {
		l.Info("before-" + itoa(round))
		l.Info(strings.Repeat("q", 500))
		l.Info("after-" + itoa(round))
	}
	l.Flush()

	got := readFile(t, path)
	for round := 0; round < 5; round++ {
		if !strings.Contains(got, "before-"+itoa(round)) {
			t.Errorf("before-%d missing", round)
		}
		if !strings.Contains(got, "after-"+itoa(round)) {
			t.Errorf("after-%d missing", round)
		}
	}
}

func TestColorWrapping(t *testing.T) {
	l, path := newFileLogger(t, 256, true)
	l.Warn("colored")
	l.Flush()

	got := readFile(t, path)
	// The timestamp is wrapped by the logger; the level tag brings its own
	// colour from the Warn method, so the two are not contiguous.
	if !strings.Contains(got, ansiGray+strings.TrimSuffix(readTS(t, got), ansiReset)+ansiReset) {
		t.Errorf("timestamp not wrapped in grey: %q", got)
	}
	if !strings.Contains(got, "[WARN] ") {
		t.Errorf("level tag missing: %q", got)
	}
	if !strings.HasSuffix(got, "colored\n") {
		t.Errorf("entry should be newline terminated: %q", got)
	}
}

func readTS(t *testing.T, line string) string {
	t.Helper()
	i := strings.Index(line, ansiGray)
	if i < 0 {
		t.Fatalf("no grey escape in %q", line)
	}
	rest := line[i+len(ansiGray):]
	j := strings.Index(rest, ansiReset)
	if j < 0 {
		t.Fatalf("no reset escape in %q", line)
	}
	return rest[:j]
}

// entryLen must exactly match what writeEntry produces, otherwise reserve
// under- or over-allocates and entries corrupt each other.
func TestEntryLenMatchesWriteEntry(t *testing.T) {
	for _, color := range []bool{false, true} {
		l := &Logger{Color: color}
		for _, ts := range []string{"2026-10-02 12:00:00", "x"} {
			for _, msg := range []string{"", "short", strings.Repeat("m", 300)} {
				want := uint32(len(l.writeEntry(nil, ts, []byte(msg))))
				if got := l.entryLen(ts, []byte(msg)); got != want {
					t.Errorf("color=%v ts=%q len(msg)=%d: entryLen=%d, writeEntry produced %d",
						color, ts, len(msg), got, want)
				}
			}
		}
	}
}

// Concurrent writers must not interleave within an entry or lose the position.
func TestConcurrentWriters(t *testing.T) {
	l, path := newFileLogger(t, 4096, false)

	var wg sync.WaitGroup
	for w := 0; w < 8; w++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for i := 0; i < 100; i++ {
				l.Info("w" + itoa(id) + "-" + itoa(i))
			}
		}(w)
	}
	wg.Wait()
	l.Flush()

	got := readFile(t, path)
	lines := strings.Split(strings.TrimRight(got, "\n"), "\n")
	if len(lines) != 800 {
		t.Fatalf("expected 800 lines, got %d", len(lines))
	}
	for _, ln := range lines {
		if !strings.Contains(ln, "[INFO] ") {
			t.Fatalf("torn entry: %q", ln)
		}
	}
}

func TestLevelFiltering(t *testing.T) {
	l, path := newFileLogger(t, 4096, false)
	l.LogLevel = Warn

	l.Debug("nope")
	l.Info("nope")
	l.Warn("yes")
	l.Error("also")
	l.Flush()

	got := readFile(t, path)
	if strings.Contains(got, "nope") {
		t.Errorf("filtered entries leaked: %q", got)
	}
	if !strings.Contains(got, "[WARN] yes") || !strings.Contains(got, "[ERROR] also") {
		t.Errorf("expected entries missing: %q", got)
	}
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(bytes.TrimRight(b, "\x00"))
}

func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	var b []byte
	for i > 0 {
		b = append([]byte{byte('0' + i%10)}, b...)
		i /= 10
	}
	return string(b)
}

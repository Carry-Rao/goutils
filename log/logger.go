package log

import (
	"os"
	"sync"
)

// Logger writes leveled, buffered entries to a file. Safe for concurrent use;
// do not copy it after first use.
type Logger struct {
	File     *os.File
	LogLevel LogLevel
	Buffer   []byte
	Len      uint32
	Pos      uint32
	Color    bool

	// mu guards Pos and Buffer. Without it a Flush racing an append can write
	// a region that was reserved but not yet filled.
	mu sync.Mutex
}

package log

import "os"

type Logger struct {
	File     *os.File
	LogLevel LogLevel
	Buffer   []byte
	Len      uint
	Color    bool
}

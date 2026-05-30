package log

import "os"

type Logger struct {
	File     *os.File
	LogLevel LogLevel
	Buffer   string
	Len      uint
	Color    bool
}

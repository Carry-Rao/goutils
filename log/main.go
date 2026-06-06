package log

import "sync/atomic"

func (l *Logger) addLog(log Log) {
	pos := atomic.AddUint32(&l.Pos, uint32(len(log.LogInfo)))
	length := uint32(len(log.LogInfo))
	end := pos + length
	if end > l.Len {
		l.Flush()
		pos = 0
		end = length
	}
	copy(l.Buffer[pos:end], log.LogInfo)
}

func (l *Logger) Flush() {
	if len(l.Buffer) > 0 {
		l.File.Write(l.Buffer)
		l.Buffer = l.Buffer[:0]
	}
}

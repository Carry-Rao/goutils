package log

func (l *Logger) addLog(log Log) {
	l.Buffer = append(l.Buffer, []byte(log.LogInfo)...)
	if len(l.Buffer) >= int(l.Len) {
		l.File.Write(l.Buffer)
		l.Buffer = l.Buffer[:0]
	}
}

func (l *Logger) Flush() {
	if len(l.Buffer) > 0 {
		l.File.Write(l.Buffer)
		l.Buffer = l.Buffer[:0]
	}
}

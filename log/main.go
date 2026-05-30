package log

func (l *Logger) addLog(log Log) {
	l.Buffer += log.LogInfo
	if len(l.Buffer) >= int(l.Len) {
		l.File.WriteString(l.Buffer)
		l.Buffer = ""
	}
}

func (l *Logger) Flush() {
	if l.Buffer != "" {
		l.File.WriteString(l.Buffer)
		l.Buffer = ""
	}
}

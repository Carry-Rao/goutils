package log

func (l *Logger) Debug(info string) {
	if l.LogLevel <= Debug {
		if l.Color {
			l.addLog(Log{Debug, "\033[32m[DEBUG]\033[0m" + info + "\n"})
		} else {
			l.addLog(Log{Debug, "[DEBUG]" + info + "\n"})
		}
	}
}

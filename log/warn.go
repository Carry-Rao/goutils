package log

func (l *Logger) Warn(info string) {
	if l.LogLevel <= Warn {
		if l.Color {
			l.addLog(Log{Warn, "\033[33m[WARN] \033[0m" + info + "\n"})
		} else {
			l.addLog(Log{Warn, "[WARN] " + info + "\n"})
		}
	}
}

package log

func (l *Logger) Error(info string) {
	if l.LogLevel <= Error {
		if l.Color {
			l.addLog(Log{Error, "\033[31m[ERROR] \033[0m" + info + "\n"})
		} else {
			l.addLog(Log{Error, "[ERROR] " + info + "\n"})
		}
	}
}

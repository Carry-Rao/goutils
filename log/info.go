package log

func (l *Logger) Info(info string) {
	if l.LogLevel <= Info {
		if l.Color {
			l.addLog(Log{Info, "\033[34m[INFO]\033[0m" + info + "\n"})
		} else {
			l.addLog(Log{Info, "[INFO]" + info + "\n"})
		}
	}
}

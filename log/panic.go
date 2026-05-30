package log

import "os"

func (l *Logger) Panic(info string) {
	if l.LogLevel <= Panic {
		if l.Color {
			l.addLog(Log{Panic, "\033[41;37m[PANIC]\033[0m" + info + "\n"})
		} else {
			l.addLog(Log{Panic, "[PANIC]" + info + "\n"})
		}
		l.Flush()
		panic(info)
	}
}

func (l *Logger) Fatal(info string) {
	if l.LogLevel <= Panic {
		if l.Color {
			l.addLog(Log{Panic, "\033[41;37m[FATAL]\033[0m" + info + "\n"})
		} else {
			l.addLog(Log{Panic, "[FATAL]" + info + "\n"})
		}
		l.Flush()
		os.Exit(1)
	}
}

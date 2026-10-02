package log

import "os"

func (l *Logger) Panic(info string) {
	if l.LogLevel <= Panic {
		if l.Color {
			l.addLog(Log{Panic, "\033[41;37m[FATAL]\033[0m " + info})
		} else {
			l.addLog(Log{Panic, "[FATAL] " + info})
		}
		l.Flush()
		panic(info)
	}
}

func (l *Logger) Fatal(info string) {
	if l.LogLevel <= Panic {
		if l.Color {
			l.addLog(Log{Panic, "\033[41;37m[FATAL]\033[0m " + info})
		} else {
			l.addLog(Log{Panic, "[FATAL] " + info})
		}
		l.Flush()
		os.Exit(1)
	}
}

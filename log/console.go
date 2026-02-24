package log

import (
	"fmt"
	"os"
	"time"
)

type Console struct {
	IsStdout bool
	Level    Level
}

func NewConsole(stdout bool) *Console {
	return &Console{IsStdout: stdout, Level: 1}
}

func (c *Console) Debug(format string, p ...any) {
	if c.Level > 0 {
		return
	}
	if c.IsStdout {
		fmt.Print("\033[32m[Debug] " + time.Now().Format("2006-01-02 15:04:05") + " \033[0m")
		fmt.Printf(format, p...)
	} else {
		fmt.Fprint(os.Stderr, "\033[32m[Debug] "+time.Now().Format("2006-01-02 15:04:05")+" \033[0m")
		fmt.Fprintf(os.Stderr, format, p...)
		fmt.Fprintln(os.Stderr)
	}
}

func (c *Console) Info(format string, p ...any) {
	if c.Level > 1 {
		return
	}
	if c.IsStdout {
		fmt.Print("\033[32m[Info] " + time.Now().Format("2006-01-02 15:04:05") + " \033[0m")
		fmt.Printf(format, p...)
		fmt.Println()
	} else {
		fmt.Fprint(os.Stderr, "\033[34m[Info] "+time.Now().Format("2006-01-02 15:04:05")+" \033[0m")
		fmt.Fprintf(os.Stderr, format, p...)
		fmt.Fprintln(os.Stderr)
	}
}

func (c *Console) Warn(format string, p ...any) {
	if c.Level > 2 {
		return
	}
	if c.IsStdout {
		fmt.Print("\033[33m[Warn] " + time.Now().Format("2006-01-02 15:04:05") + " \033[0m")
		fmt.Printf(format, p...)
		fmt.Println()
	} else {
		fmt.Fprint(os.Stderr, "\033[33m[Warn] "+time.Now().Format("2006-01-02 15:04:05")+" \033[0m")
		fmt.Fprintf(os.Stderr, format, p...)
		fmt.Fprintln(os.Stderr)
	}
}

func (c *Console) Error(format string, p ...any) {
	if c.Level > 3 {
		return
	}
	if c.IsStdout {
		fmt.Print("\033[31m[Error] " + time.Now().Format("2006-01-02 15:04:05") + " \033[0m")
		fmt.Printf(format, p...)
		fmt.Println()
	} else {
		fmt.Fprint(os.Stderr, "\033[31m[Error] "+time.Now().Format("2006-01-02 15:04:05")+" \033[0m")
		fmt.Fprintf(os.Stderr, format, p...)
		fmt.Fprintln(os.Stderr)
	}
}

func (c *Console) Fatal(format string, p ...any) {
	if c.Level > 4 {
		return
	}
	if c.IsStdout {
		fmt.Print("\033[35m[Fatal] " + time.Now().Format("2006-01-02 15:04:05") + " \033[0m")
		fmt.Printf(format, p...)
		fmt.Println()
	} else {
		fmt.Fprint(os.Stderr, "\033[35m[Fatal] "+time.Now().Format("2006-01-02 15:04:05")+" \033[0m")
		fmt.Printf(format, p...)
		fmt.Fprintln(os.Stderr)
	}
	os.Exit(1)
}

func (c *Console) Panic(format string, p ...any) {
	if c.Level > 5 {
		return
	}
	if c.IsStdout {
		fmt.Print("\033[35m[Panic] " + time.Now().Format("2006-01-02 15:04:05") + " \033[0m")
		fmt.Printf(format, p...)
		fmt.Println()
	} else {
		fmt.Fprint(os.Stderr, "\033[35m[Panic] "+time.Now().Format("2006-01-02 15:04:05")+" \033[0m")
		fmt.Fprintf(os.Stderr, format, p...)
		fmt.Fprintln(os.Stderr)
	}
	panic(fmt.Sprintf(format, p...))
}

func (c *Console) SetLevel(level Level) {
	c.Level = level
}

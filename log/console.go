package log

import (
	"fmt"
	"os"
	"time"
)

type Console struct {
	IsStdout bool
	Level    int
}

func NewConsole(stdout bool) *Console {
	return &Console{IsStdout: stdout, Level: 1}
}

func (c *Console) Debug(p string) {
	if c.Level > 0 {
		return
	}
	if c.IsStdout {
		fmt.Print("\033[32m[Debug] " + time.Now().Format("2006-01-02 15:04:05") + " \033[0m")
		fmt.Println(p)
	} else {
		fmt.Print("\033[32m[Debug] " + time.Now().Format("2006-01-02 15:04:05") + " \033[0m")
		fmt.Println(p)
	}
}

func (c *Console) Info(p string) {
	if c.Level > 1 {
		return
	}
	if c.IsStdout {
		fmt.Print("\033[32m[Info] " + time.Now().Format("2006-01-02 15:04:05") + " \033[0m")
		fmt.Println(p)
	} else {
		fmt.Print("\033[34m[Info] " + time.Now().Format("2006-01-02 15:04:05") + " \033[0m")
		fmt.Println(p)
	}
}

func (c *Console) Warn(p string) {
	if c.Level > 2 {
		return
	}
	if c.IsStdout {
		fmt.Print("\033[33m[Warn] " + time.Now().Format("2006-01-02 15:04:05") + " \033[0m")
		fmt.Println(p)
	} else {
		fmt.Print("\033[33m[Warn] " + time.Now().Format("2006-01-02 15:04:05") + " \033[0m")
		fmt.Println(p)
	}
}

func (c *Console) Error(p string) {
	if c.Level > 3 {
		return
	}
	if c.IsStdout {
		fmt.Print("\033[31m[Error] " + time.Now().Format("2006-01-02 15:04:05") + " \033[0m")
		fmt.Println(p)
	} else {
		fmt.Print("\033[31m[Error] " + time.Now().Format("2006-01-02 15:04:05") + " \033[0m")
		fmt.Println(p)
	}
}

func (c *Console) Fatal(p string) {
	if c.Level > 4 {
		return
	}
	if c.IsStdout {
		fmt.Print("\033[35m[Fatal] " + time.Now().Format("2006-01-02 15:04:05") + " \033[0m")
		fmt.Println(p)
	} else {
		fmt.Print("\033[35m[Fatal] " + time.Now().Format("2006-01-02 15:04:05") + " \033[0m")
		fmt.Println(p)
	}
	os.Exit(1)
}

func (c *Console) Panic(p string) {
	if c.Level > 5 {
		return
	}
	if c.IsStdout {
		fmt.Print("\033[35m[Panic] " + time.Now().Format("2006-01-02 15:04:05") + " \033[0m")
		fmt.Println(p)
	} else {
		fmt.Print("\033[35m[Panic] " + time.Now().Format("2006-01-02 15:04:05") + " \033[0m")
		fmt.Println(p)
	}
	panic(p)
}

func (c *Console) SetLevel(level int) {
	c.Level = level
}

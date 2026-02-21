package main

import "github.com/Carry-Rao/goutils/log"

func main() {
	logger := log.NewConsole(false)
	logger.SetLevel(log.Debug)
	logger.Debug("Hello, world!")
	logger.Info("Hello, world!")
	logger.Warn("Hello, world!")
	logger.Error("Hello, world!")
	logger.Fatal("Hello, world!")
}

package main

import (
	"log"
	"os"
)

// logDebug writes formatted text to debug.log in the current directory
func logDebug(format string, v ...interface{}) {
	f, err := os.OpenFile("debug.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	defer f.Close()

	logger := log.New(f, "DEBUG: ", log.LstdFlags)
	logger.Printf(format, v...)
}

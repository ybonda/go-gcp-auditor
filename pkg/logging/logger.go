package logging

import (
	"fmt"
	"os"
	"time"
)

type LogLevel int

const (
	LevelError LogLevel = iota
	LevelInfo
	LevelDebug
)

type Logger struct {
	verbose bool
}

func NewLogger(verbose bool) *Logger {
	return &Logger{
		verbose: verbose,
	}
}

func (l *Logger) Info(format string, args ...interface{}) {
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	fmt.Printf("%s [INFO] %s\n", timestamp, fmt.Sprintf(format, args...))
}

func (l *Logger) Debug(format string, args ...interface{}) {
	if l.verbose {
		timestamp := time.Now().Format("2006-01-02 15:04:05")
		fmt.Printf("%s [DEBUG] %s\n", timestamp, fmt.Sprintf(format, args...))
	}
}

func (l *Logger) Error(format string, args ...interface{}) {
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	fmt.Fprintf(os.Stderr, "%s [ERROR] %s\n", timestamp, fmt.Sprintf(format, args...))
}

func (l *Logger) Progress(format string, args ...interface{}) {
	fmt.Printf(format+"\n", args...)
}

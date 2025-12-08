package logger

import (
	"fmt"
	"log"
	"os"
	"time"
)

type Logger struct {
	debug  bool
	logger *log.Logger
}

func New(debug bool) *Logger {
	return &Logger{
		debug:  debug,
		logger: log.New(os.Stdout, "", 0),
	}
}

func (l *Logger) Info(format string, args ...interface{}) {
	l.log("INFO", format, args...)
}

func (l *Logger) Error(format string, args ...interface{}) {
	l.log("ERROR", format, args...)
}

func (l *Logger) Debug(format string, args ...interface{}) {
	if l.debug {
		l.log("DEBUG", format, args...)
	}
}

func (l *Logger) Warn(format string, args ...interface{}) {
	l.log("WARN", format, args...)
}

func (l *Logger) log(level string, format string, args ...interface{}) {
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	message := fmt.Sprintf(format, args...)
	l.logger.Printf("[%s] %s: %s", timestamp, level, message)
}

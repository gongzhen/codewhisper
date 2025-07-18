package utils

import (
	"fmt"
	"log"
	"strings"

	"github.com/gongzhen/codewhisper-go/pkg/config"
)

// LogLevel represents logging levels
type LogLevel int

const (
    DEBUG LogLevel = iota
    INFO
    WARNING
    ERROR
)

// Logger is our custom logger with colored output
type Logger struct {
    level  LogLevel
    prefix string
}

// NewLogger creates a new logger instance
func NewLogger() *Logger {
    levelStr := config.GetEnv(config.EnvLogLevel, "INFO")
    level := parseLogLevel(levelStr)
    
    // Set up the standard logger to not include date/time (we'll add our own)
    log.SetFlags(0)
    
    return &Logger{
        level:  level,
        prefix: "\033[35mCODEWHISPER\033[0m:     ", // Purple color for CodeWhisper
    }
}

func parseLogLevel(level string) LogLevel {
    switch strings.ToUpper(level) {
    case "DEBUG":
        return DEBUG
    case "WARNING", "WARN":
        return WARNING
    case "ERROR":
        return ERROR
    default:
        return INFO
    }
}

// Debug logs a debug message
func (l *Logger) Debug(format string, v ...interface{}) {
    if l.level <= DEBUG {
        l.output("DEBUG", format, v...)
    }
}

// Info logs an info message
func (l *Logger) Info(format string, v ...interface{}) {
    if l.level <= INFO {
        l.output("INFO", format, v...)
    }
}

// Warning logs a warning message
func (l *Logger) Warning(format string, v ...interface{}) {
    if l.level <= WARNING {
        l.output("WARN", format, v...)
    }
}

// Error logs an error message
func (l *Logger) Error(format string, v ...interface{}) {
    if l.level <= ERROR {
        l.output("ERROR", format, v...)
    }
}

func (l *Logger) output(level, format string, v ...interface{}) {
    message := fmt.Sprintf(format, v...)
    
    // Add log level indicator for non-INFO messages
    if level != "INFO" {
        fmt.Printf("%s[%s] %s\n", l.prefix, level, message)
    } else {
        fmt.Printf("%s%s\n", l.prefix, message)
    }
}

// Global logger instance
var Log = NewLogger()

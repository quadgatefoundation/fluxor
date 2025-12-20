package core

import (
	"fmt"
	"log"
	"os"
)

// Logger provides structured logging capabilities
// This abstraction allows swapping logging implementations
type Logger interface {
	// Error logs an error message
	Error(args ...interface{})
	
	// Errorf logs a formatted error message
	Errorf(format string, args ...interface{})
	
	// Warn logs a warning message
	Warn(args ...interface{})
	
	// Warnf logs a formatted warning message
	Warnf(format string, args ...interface{})
	
	// Info logs an informational message
	Info(args ...interface{})
	
	// Infof logs a formatted informational message
	Infof(format string, args ...interface{})
	
	// Debug logs a debug message
	Debug(args ...interface{})
	
	// Debugf logs a formatted debug message
	Debugf(format string, args ...interface{})
}

// defaultLogger implements Logger using Go's standard log package
// Can be swapped with other logging implementations (e.g., structured loggers)
type defaultLogger struct {
	errorLogger *log.Logger
	warnLogger  *log.Logger
	infoLogger  *log.Logger
	debugLogger *log.Logger
}

// NewDefaultLogger creates a new default logger implementation
func NewDefaultLogger() Logger {
	return &defaultLogger{
		errorLogger: log.New(os.Stderr, "[ERROR] ", log.LstdFlags|log.Lshortfile),
		warnLogger:  log.New(os.Stderr, "[WARN] ", log.LstdFlags|log.Lshortfile),
		infoLogger:  log.New(os.Stdout, "[INFO] ", log.LstdFlags|log.Lshortfile),
		debugLogger: log.New(os.Stdout, "[DEBUG] ", log.LstdFlags|log.Lshortfile),
	}
}

// Error logs an error message
func (l *defaultLogger) Error(args ...interface{}) {
	l.errorLogger.Output(3, fmt.Sprint(args...))
}

// Errorf logs a formatted error message
func (l *defaultLogger) Errorf(format string, args ...interface{}) {
	l.errorLogger.Output(3, fmt.Sprintf(format, args...))
}

// Warn logs a warning message
func (l *defaultLogger) Warn(args ...interface{}) {
	l.warnLogger.Output(3, fmt.Sprint(args...))
}

// Warnf logs a formatted warning message
func (l *defaultLogger) Warnf(format string, args ...interface{}) {
	l.warnLogger.Output(3, fmt.Sprintf(format, args...))
}

// Info logs an informational message
func (l *defaultLogger) Info(args ...interface{}) {
	l.infoLogger.Output(3, fmt.Sprint(args...))
}

// Infof logs a formatted informational message
func (l *defaultLogger) Infof(format string, args ...interface{}) {
	l.infoLogger.Output(3, fmt.Sprintf(format, args...))
}

// Debug logs a debug message
func (l *defaultLogger) Debug(args ...interface{}) {
	l.debugLogger.Output(3, fmt.Sprint(args...))
}

// Debugf logs a formatted debug message
func (l *defaultLogger) Debugf(format string, args ...interface{}) {
	l.debugLogger.Output(3, fmt.Sprintf(format, args...))
}


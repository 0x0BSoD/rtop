package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
)

// LogLevel represents the severity level of a log message
type LogLevel int

// Log levels
const (
	DEBUG LogLevel = iota
	INFO
	WARN
	ERROR
	FATAL
)

// Logger provides structured logging with levels
type Logger struct {
	level     LogLevel
	logFile   *os.File
	logger    *log.Logger
	console   bool
	logToFile bool
}

// Global logger instance
var rtopLogger *Logger

// InitLogging initializes the logging system
func InitLogging(logLevel string, logToConsole bool, logFilePath string) {
	var level LogLevel

	// Parse log level
	switch strings.ToUpper(logLevel) {
	case "DEBUG":
		level = DEBUG
	case "INFO":
		level = INFO
	case "WARN":
		level = WARN
	case "ERROR":
		level = ERROR
	case "FATAL":
		level = FATAL
	default:
		level = INFO
	}

	var logFile *os.File
	var err error
	logToFile := false

	// Setup log file if path is provided
	if logFilePath != "" {
		// Create log directory if it doesn't exist
		logDir := filepath.Dir(logFilePath)
		if _, err := os.Stat(logDir); os.IsNotExist(err) {
			if err := os.MkdirAll(logDir, 0755); err != nil {
				fmt.Fprintf(os.Stderr, "Error creating log directory %s: %v\n", logDir, err)
			}
		}

		// Open log file
		logFile, err = os.OpenFile(logFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error opening log file: %v\n", err)
		} else {
			logToFile = true
		}
	}

	// Create logger with appropriate writers
	var writer io.Writer
	if logToConsole && logToFile {
		writer = io.MultiWriter(os.Stderr, logFile)
	} else if logToConsole {
		writer = os.Stderr
	} else if logToFile {
		writer = logFile
	} else {
		// If no output is specified, use stderr as fallback
		writer = os.Stderr
		logToConsole = true
	}

	logger := log.New(writer, "rtop: ", log.LstdFlags|log.Lmicroseconds)

	rtopLogger = &Logger{
		level:     level,
		logFile:   logFile,
		logger:    logger,
		console:   logToConsole,
		logToFile: logToFile,
	}

	rtopLogger.Info("Logging initialized at level: %s", logLevel)
}

// Close closes any resources used by the logger
func (l *Logger) Close() {
	if l.logFile != nil {
		l.logFile.Close()
	}
}

// log logs a message with the specified level
func (l *Logger) log(level LogLevel, format string, v ...interface{}) {
	if level < l.level {
		return
	}

	// Get level string
	var levelStr string
	switch level {
	case DEBUG:
		levelStr = "DEBUG"
	case INFO:
		levelStr = "INFO"
	case WARN:
		levelStr = "WARN"
	case ERROR:
		levelStr = "ERROR"
	case FATAL:
		levelStr = "FATAL"
	}

	// Format message with timestamp
	message := fmt.Sprintf(format, v...)
	l.logger.Printf("[%s] %s", levelStr, message)

	// Exit if fatal
	if level == FATAL {
		os.Exit(1)
	}
}

// Debug logs a debug message
func (l *Logger) Debug(format string, v ...interface{}) {
	l.log(DEBUG, format, v...)
}

// Info logs an informational message
func (l *Logger) Info(format string, v ...interface{}) {
	l.log(INFO, format, v...)
}

// Warn logs a warning message
func (l *Logger) Warn(format string, v ...interface{}) {
	l.log(WARN, format, v...)
}

// Error logs an error message
func (l *Logger) Error(format string, v ...interface{}) {
	l.log(ERROR, format, v...)
}

// Fatal logs a fatal message and exits
func (l *Logger) Fatal(format string, v ...interface{}) {
	l.log(FATAL, format, v...)
}

// LogConnectionAttempt logs SSH connection attempts with relevant details
func (l *Logger) LogConnectionAttempt(username string, host string, port int) {
	l.Info("Connection attempt to %s@%s:%d", username, host, port)
}

// LogConnectionSuccess logs successful SSH connections
func (l *Logger) LogConnectionSuccess(username string, host string, port int) {
	l.Info("Successfully connected to %s@%s:%d", username, host, port)
}

// LogCommandExecution logs command executions over SSH
func (l *Logger) LogCommandExecution(command string, host string) {
	l.Debug("Executing command on %s: %s", host, command)
}

// LogStats logs a summary of the collected stats
func (l *Logger) LogStats(hostname string, load1 string, memUsedPercent float64, cpuIdlePercent float64) {
	l.Debug("Stats for %s: Load: %s, Mem Used: %.2f%%, CPU Idle: %.2f%%",
		hostname, load1, memUsedPercent, cpuIdlePercent)
}

// Debug convenience function for global logger
func Debug(format string, v ...interface{}) {
	if rtopLogger != nil {
		rtopLogger.Debug(format, v...)
	}
}

// Info convenience function for global logger
func Info(format string, v ...interface{}) {
	if rtopLogger != nil {
		rtopLogger.Info(format, v...)
	}
}

// Warn convenience function for global logger
func Warn(format string, v ...interface{}) {
	if rtopLogger != nil {
		rtopLogger.Warn(format, v...)
	}
}

// Error convenience function for global logger
func Error(format string, v ...interface{}) {
	if rtopLogger != nil {
		rtopLogger.Error(format, v...)
	}
}

// Fatal convenience function for global logger
func Fatal(format string, v ...interface{}) {
	if rtopLogger != nil {
		rtopLogger.Fatal(format, v...)
	} else {
		// Ensure we exit even if logger is not initialized
		fmt.Fprintf(os.Stderr, "FATAL: "+format+"\n", v...)
		os.Exit(1)
	}
}

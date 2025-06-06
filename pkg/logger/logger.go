package logger

import (
	"fmt"
	"log"
	"os"

	"github.com/fatih/color"
)

var defaultLogger *Logger

// Logger represents the application logger
type Logger struct {
	verbose bool
	quiet   bool
}

// Init initializes the default logger
func Init() {
	defaultLogger = New(false, false)
}

// New creates a new logger instance
func New(verbose, quiet bool) *Logger {
	return &Logger{
		verbose: verbose,
		quiet:   quiet,
	}
}

// SetVerbose sets verbose mode for the default logger
func SetVerbose(verbose bool) {
	if defaultLogger != nil {
		defaultLogger.verbose = verbose
	}
}

// SetQuiet sets quiet mode for the default logger
func SetQuiet(quiet bool) {
	if defaultLogger != nil {
		defaultLogger.quiet = quiet
	}
}

// Info logs an info message using the default logger
func Info(msg string, args ...interface{}) {
	if defaultLogger != nil {
		defaultLogger.Info(msg, args...)
	}
}

// Success logs a success message using the default logger
func Success(msg string, args ...interface{}) {
	if defaultLogger != nil {
		defaultLogger.Success(msg, args...)
	}
}

// Warning logs a warning message using the default logger
func Warning(msg string, args ...interface{}) {
	if defaultLogger != nil {
		defaultLogger.Warning(msg, args...)
	}
}

// Error logs an error message using the default logger
func Error(msg string, args ...interface{}) {
	if defaultLogger != nil {
		defaultLogger.Error(msg, args...)
	}
}

// Debug logs a debug message using the default logger
func Debug(msg string, args ...interface{}) {
	if defaultLogger != nil {
		defaultLogger.Debug(msg, args...)
	}
}

// Fatal logs a fatal error and exits using the default logger
func Fatal(msg string, args ...interface{}) {
	if defaultLogger != nil {
		defaultLogger.Fatal(msg, args...)
	} else {
		color.Red("FATAL: " + fmt.Sprintf(msg, args...))
		os.Exit(1)
	}
}

// Info logs an info message
func (l *Logger) Info(msg string, args ...interface{}) {
	if !l.quiet {
		color.Blue("INFO: " + fmt.Sprintf(msg, args...))
	}
}

// Success logs a success message
func (l *Logger) Success(msg string, args ...interface{}) {
	if !l.quiet {
		color.Green("SUCCESS: " + fmt.Sprintf(msg, args...))
	}
}

// Warning logs a warning message
func (l *Logger) Warning(msg string, args ...interface{}) {
	if !l.quiet {
		color.Yellow("WARNING: " + fmt.Sprintf(msg, args...))
	}
}

// Error logs an error message
func (l *Logger) Error(msg string, args ...interface{}) {
	color.Red("ERROR: " + fmt.Sprintf(msg, args...))
}

// Debug logs a debug message (only when verbose)
func (l *Logger) Debug(msg string, args ...interface{}) {
	if l.verbose && !l.quiet {
		color.Cyan("DEBUG: " + fmt.Sprintf(msg, args...))
	}
}

// Fatal logs a fatal error and exits
func (l *Logger) Fatal(msg string, args ...interface{}) {
	color.Red("FATAL: " + fmt.Sprintf(msg, args...))
	os.Exit(1)
}

// Printf implements basic printf functionality
func (l *Logger) Printf(format string, args ...interface{}) {
	if !l.quiet {
		log.Printf(format, args...)
	}
}

// Println implements basic println functionality
func (l *Logger) Println(args ...interface{}) {
	if !l.quiet {
		log.Println(args...)
	}
}

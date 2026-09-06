package logging

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Logger provides application-wide logging with file output
type Logger struct {
	file       *os.File
	errorFile  *os.File
	logger     *log.Logger
	errorLog   *log.Logger
	verbose    bool
	mu         sync.Mutex
	logFile    string
	errorLogFile string
	stderrFile string
}

var (
	globalLogger   *Logger
	once           sync.Once
	projectLoggers sync.Map // map[projectID]*Logger
)

// Initialize creates the global logger instance
// logDir: directory where log files will be written
// verbose: enable debug/verbose logging
func Initialize(logDir string, verbose bool) (*Logger, error) {
	var err error
	once.Do(func() {
		globalLogger, err = newLogger(logDir, verbose)
	})
	return globalLogger, err
}

// GetLogger returns the global logger instance
func GetLogger() *Logger {
	return globalLogger
}

// newLogger creates a new logger instance
func newLogger(logDir string, verbose bool) (*Logger, error) {
	// Create log directory if it doesn't exist
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create log directory: %w", err)
	}

	// Use fixed filenames without timestamps
	logFile := filepath.Join(logDir, "wee.log")
	errorLogFile := filepath.Join(logDir, "errors.log")
	stderrFile := filepath.Join(logDir, "sdk_stderr.log")

	// Create log file (append mode)
	file, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to create log file: %w", err)
	}

	// Create error log file (append mode)
	errorFile, err := os.OpenFile(errorLogFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		file.Close()
		return nil, fmt.Errorf("failed to create error log file: %w", err)
	}

	// Create logger without timestamp (LstdFlags removed, using Lshortfile only)
	logger := log.New(file, "", log.Lshortfile)
	errorLog := log.New(errorFile, "", log.Lshortfile)

	l := &Logger{
		file:         file,
		errorFile:    errorFile,
		logger:       logger,
		errorLog:     errorLog,
		verbose:      verbose,
		logFile:      logFile,
		errorLogFile: errorLogFile,
		stderrFile:   stderrFile,
	}

	// Log initialization
	l.Info("Logger initialized (verbose=%v)", verbose)
	l.Info("Log file: %s", logFile)
	l.Info("Error log file: %s", errorLogFile)
	l.Info("SDK stderr file: %s", stderrFile)

	return l, nil
}

// Debug logs a debug message (only when verbose is enabled)
func (l *Logger) Debug(format string, args ...interface{}) {
	if l.verbose {
		l.log("DEBUG", format, args...)
	}
}

// Info logs an informational message
func (l *Logger) Info(format string, args ...interface{}) {
	l.log("INFO", format, args...)
}

// Warning logs a warning message
func (l *Logger) Warning(format string, args ...interface{}) {
	l.log("WARNING", format, args...)
}

// Error logs an error message
func (l *Logger) Error(format string, args ...interface{}) {
	l.log("ERROR", format, args...)
	// Also log to error log file
	if l.errorLog != nil {
		l.mu.Lock()
		defer l.mu.Unlock()
		msg := fmt.Sprintf(format, args...)
		l.errorLog.Printf("[ERROR] %s", msg)
	}
}

// log is the internal logging method
func (l *Logger) log(level string, format string, args ...interface{}) {
	if l == nil || l.logger == nil {
		return
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	msg := fmt.Sprintf(format, args...)
	l.logger.Printf("[%s] %s", level, msg)

	// Broadcast to any active subscribers (non-blocking)
	broadcaster := GetBroadcaster()
	if broadcaster.HasSubscribers() {
		broadcaster.Broadcast(LogEntry{
			Level:     level,
			Message:   msg,
			Timestamp: time.Now(),
		})
	}
}

// StandardLogWriter returns the file destination for standard-library logs.
func (l *Logger) StandardLogWriter() io.Writer {
	return l.file
}

// GetLogFilePath returns the path to the main log file
func (l *Logger) GetLogFilePath() string {
	if l == nil {
		return ""
	}
	return l.logFile
}

// GetErrorLogFilePath returns the path to the error log file
func (l *Logger) GetErrorLogFilePath() string {
	if l == nil {
		return ""
	}
	return l.errorLogFile
}

// GetStderrFilePath returns the path to the stderr log file
func (l *Logger) GetStderrFilePath() string {
	if l == nil {
		return ""
	}
	return l.stderrFile
}

// RedirectStderr redirects os.Stderr to the stderr log file
// This captures all stderr output (including from claude-agent-sdk-go) to a file
func (l *Logger) RedirectStderr() error {
	if l == nil {
		return fmt.Errorf("logger not initialized")
	}

	// Create stderr log file
	stderrFile, err := os.OpenFile(l.stderrFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("failed to create stderr log file: %w", err)
	}

	// Save original stderr
	originalStderr := os.Stderr

	// Create a multi-writer that writes to both the file and original stderr
	// This way we can still see errors in the console if needed
	multiWriter := io.MultiWriter(stderrFile, originalStderr)

	// Create a pipe
	reader, writer, err := os.Pipe()
	if err != nil {
		return fmt.Errorf("failed to create pipe: %w", err)
	}

	// Set os.Stderr to the writer end of the pipe
	os.Stderr = writer

	// Start goroutine to copy from pipe to both file and original stderr
	go func() {
		_, _ = io.Copy(multiWriter, reader)
	}()

	l.Info("Stderr redirected to: %s", l.stderrFile)
	return nil
}

// Close closes the log files
func (l *Logger) Close() error {
	if l == nil {
		return nil
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	l.Info("Logger shutting down")

	// Close main log file
	var err error
	if l.file != nil {
		if ferr := l.file.Close(); ferr != nil {
			err = ferr
		}
		l.file = nil
	}

	// Close error log file
	if l.errorFile != nil {
		if ferr := l.errorFile.Close(); ferr != nil && err == nil {
			err = ferr
		}
		l.errorFile = nil
	}

	return err
}

// IsVerbose returns whether verbose logging is enabled
func (l *Logger) IsVerbose() bool {
	if l == nil {
		return false
	}
	return l.verbose
}

// Helper functions for global logger

// Debug logs a debug message to the global logger
func Debug(format string, args ...interface{}) {
	if globalLogger != nil {
		globalLogger.Debug(format, args...)
	}
}

// Info logs an info message to the global logger
func Info(format string, args ...interface{}) {
	if globalLogger != nil {
		globalLogger.Info(format, args...)
	}
}

// Warning logs a warning message to the global logger
func Warning(format string, args ...interface{}) {
	if globalLogger != nil {
		globalLogger.Warning(format, args...)
	}
}

// Error logs an error message to the global logger
func Error(format string, args ...interface{}) {
	if globalLogger != nil {
		globalLogger.Error(format, args...)
	}
}

// CreateProjectLogger creates a project-specific logger instance
// This logger writes to a separate file dedicated to the project
// projectID: unique identifier for the project
// projectType: type of project (e.g., "landingpage", "agent", etc.)
// logDir: directory where project log files will be written
// verbose: enable debug/verbose logging
func CreateProjectLogger(projectID, projectType, logDir string, verbose bool) (*Logger, error) {
	// Check if logger already exists for this project
	if existing, ok := projectLoggers.Load(projectID); ok {
		return existing.(*Logger), nil
	}

	// Create project-specific log directory
	projectLogDir := filepath.Join(logDir, projectType)
	if err := os.MkdirAll(projectLogDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create project log directory: %w", err)
	}

	// Use fixed log filenames without timestamps
	logFile := filepath.Join(projectLogDir, fmt.Sprintf("%s.log", projectID))
	errorLogFile := filepath.Join(projectLogDir, fmt.Sprintf("%s_errors.log", projectID))

	// Create log file (append mode)
	file, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to create project log file: %w", err)
	}

	// Create error log file (append mode)
	errorFile, err := os.OpenFile(errorLogFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		file.Close()
		return nil, fmt.Errorf("failed to create project error log file: %w", err)
	}

	// Create multi-writer to write to both project log and global log
	var writer io.Writer
	if globalLogger != nil && globalLogger.file != nil {
		writer = io.MultiWriter(file, globalLogger.file)
	} else {
		writer = file
	}

	// Create multi-writer for error logs
	var errorWriter io.Writer
	if globalLogger != nil && globalLogger.errorFile != nil {
		errorWriter = io.MultiWriter(errorFile, globalLogger.errorFile)
	} else {
		errorWriter = errorFile
	}

	// Create loggers with project prefix, without timestamp (removed LstdFlags)
	prefix := fmt.Sprintf("[%s] ", projectID)
	logger := log.New(writer, prefix, log.Lshortfile)
	errorLog := log.New(errorWriter, prefix, log.Lshortfile)

	l := &Logger{
		file:         file,
		errorFile:    errorFile,
		logger:       logger,
		errorLog:     errorLog,
		verbose:      verbose,
		logFile:      logFile,
		errorLogFile: errorLogFile,
	}

	// Store in project loggers map
	projectLoggers.Store(projectID, l)

	// Log initialization
	l.Info("Project logger initialized for %s (type=%s, verbose=%v)", projectID, projectType, verbose)
	l.Info("Project log file: %s", logFile)
	l.Info("Project error log file: %s", errorLogFile)

	return l, nil
}

// GetProjectLogger retrieves an existing project logger
func GetProjectLogger(projectID string) (*Logger, bool) {
	if logger, ok := projectLoggers.Load(projectID); ok {
		return logger.(*Logger), true
	}
	return nil, false
}

// CloseProjectLogger closes and removes a project logger
func CloseProjectLogger(projectID string) error {
	if logger, ok := projectLoggers.LoadAndDelete(projectID); ok {
		return logger.(*Logger).Close()
	}
	return nil
}

// ListProjectLoggers returns a list of all active project IDs with loggers
func ListProjectLoggers() []string {
	var projects []string
	projectLoggers.Range(func(key, value interface{}) bool {
		projects = append(projects, key.(string))
		return true
	})
	return projects
}

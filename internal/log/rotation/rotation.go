package rotation

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	LogDateFormat = "2006-01-02"
	LogFilePrefix = "tmixer-"
	LogFileSuffix = ".jsonl"
)

// RotateLogFile manages log file rotation based on retention duration.
// It deletes logs older than the retention period and creates/opens
// a log file for the current day.
//
// Parameters:
//   - logDir: Directory where log files are stored
//   - retention: Duration to keep old log files (e.g., 7*24*time.Hour for 7 days)
//
// Returns:
//   - *os.File: The opened log file for today
//   - error: Any error encountered during rotation or file creation
func RotateLogFile(logDir string, retention time.Duration) (*os.File, error) {
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return nil, fmt.Errorf("failed get user home dir: %w", err)
	}
	// Ensure log directory exists
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create log directory: %w", err)
	}

	// Delete old log files
	if err := deleteOldLogs(logDir, retention); err != nil {
		return nil, fmt.Errorf("failed to delete old logs: %w", err)
	}

	// Create/open log file for current day
	currentDate := time.Now().UTC().Format(LogDateFormat)
	logFileName := fmt.Sprintf("%s%s%s", LogFilePrefix, currentDate, LogFileSuffix)
	logFilePath := filepath.Join(logDir, logFileName)

	if retention >= 0 {
		file, err := os.OpenFile(logFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return nil, fmt.Errorf("failed to open log file: %w", err)
		}
		return file, nil
	} else {
		return nil, nil
	}
}

// deleteOldLogs removes log files older than the retention duration
func deleteOldLogs(logDir string, retention time.Duration) error {
	entries, err := os.ReadDir(logDir)
	if err != nil {
		return err
	}

	cutoffTime := time.Now().UTC().Truncate(24 * time.Hour).Add(-retention)

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()

		// Only process files matching our log file pattern
		if !strings.HasPrefix(name, LogFilePrefix) || !strings.HasSuffix(name, LogFileSuffix) {
			continue
		}

		// Extract date from filename (e.g., "tmixer-2026-01-06.log" -> "2026-01-06")
		dateStr := strings.TrimPrefix(name, LogFilePrefix)
		dateStr = strings.TrimSuffix(dateStr, LogFileSuffix)

		// Parse the date from the filename
		logDate, err := time.Parse(LogDateFormat, dateStr)
		if err != nil {
			// Skip files that don't match our expected date format
			continue
		}

		// Delete if older than retention period
		if logDate.Before(cutoffTime) {
			logPath := filepath.Join(logDir, name)
			if err := os.Remove(logPath); err != nil {
				return fmt.Errorf("failed to delete old log file %s: %w", name, err)
			}
		}
	}

	return nil
}

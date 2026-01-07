package log

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestRotateLogFile(t *testing.T) {
	// Create a temporary directory for test logs
	tmpDir := t.TempDir()

	// Create some old log files
	oldDate := time.Now().AddDate(0, 0, -10).Format(logDateFormat)
	oldLogPath := filepath.Join(tmpDir, logFilePrefix+oldDate+logFileSuffix)
	if err := os.WriteFile(oldLogPath, []byte("old log"), 0644); err != nil {
		t.Fatalf("Failed to create old log file: %v", err)
	}

	// Create a recent log file (should be kept)
	recentDate := time.Now().AddDate(0, 0, -2).Format(logDateFormat)
	recentLogPath := filepath.Join(tmpDir, logFilePrefix+recentDate+logFileSuffix)
	if err := os.WriteFile(recentLogPath, []byte("recent log"), 0644); err != nil {
		t.Fatalf("Failed to create recent log file: %v", err)
	}

	// Run rotation with 7 day retention
	retention := 7 * 24 * time.Hour
	file, err := RotateLogFile(tmpDir, retention)
	if err != nil {
		t.Fatalf("RotateLogFile failed: %v", err)
	}
	defer file.Close()

	// Check that old log was deleted
	if _, err := os.Stat(oldLogPath); !os.IsNotExist(err) {
		t.Errorf("Old log file should have been deleted: %s", oldLogPath)
	}

	// Check that recent log was kept
	if _, err := os.Stat(recentLogPath); err != nil {
		t.Errorf("Recent log file should have been kept: %s", recentLogPath)
	}

	// Check that today's log file was created
	todayDate := time.Now().UTC().Format(logDateFormat)
	todayLogPath := filepath.Join(tmpDir, logFilePrefix+todayDate+logFileSuffix)
	if _, err := os.Stat(todayLogPath); err != nil {
		t.Errorf("Today's log file should have been created: %s", todayLogPath)
	}

	// Verify we can write to the file
	if _, err := file.WriteString("test log entry\n"); err != nil {
		t.Errorf("Failed to write to log file: %v", err)
	}
}

func TestRotateLogFile_CreatesDirectory(t *testing.T) {
	// Use a non-existent directory
	tmpDir := filepath.Join(t.TempDir(), "nested", "log", "dir")

	file, err := RotateLogFile(tmpDir, 24*time.Hour)
	if err != nil {
		t.Fatalf("RotateLogFile should create directory: %v", err)
	}
	defer file.Close()

	// Verify directory was created
	if _, err := os.Stat(tmpDir); err != nil {
		t.Errorf("Log directory should have been created: %v", err)
	}
}

func TestRotateLogFile_IgnoresNonLogFiles(t *testing.T) {
	tmpDir := t.TempDir()

	// Create files that don't match the log pattern
	otherFiles := []string{
		"some-other-file.log",
		"tmixer-not-a-date.log",
		"random.txt",
		".hidden",
	}

	for _, name := range otherFiles {
		path := filepath.Join(tmpDir, name)
		if err := os.WriteFile(path, []byte("content"), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}
	}

	// Run rotation
	file, err := RotateLogFile(tmpDir, 24*time.Hour)
	if err != nil {
		t.Fatalf("RotateLogFile failed: %v", err)
	}
	defer file.Close()

	// Verify all non-log files are still there
	for _, name := range otherFiles {
		path := filepath.Join(tmpDir, name)
		if _, err := os.Stat(path); err != nil {
			t.Errorf("Non-log file should not have been deleted: %s", name)
		}
	}
}

func TestRotateLogFile_AppendMode(t *testing.T) {
	tmpDir := t.TempDir()

	// Create today's log file with existing content
	todayDate := time.Now().UTC().Format(logDateFormat)
	todayLogPath := filepath.Join(tmpDir, logFilePrefix+todayDate+logFileSuffix)
	existingContent := "existing log entry\n"
	if err := os.WriteFile(todayLogPath, []byte(existingContent), 0644); err != nil {
		t.Fatalf("Failed to create existing log file: %v", err)
	}

	// Open with rotation (should append, not truncate)
	file, err := RotateLogFile(tmpDir, 24*time.Hour)
	if err != nil {
		t.Fatalf("RotateLogFile failed: %v", err)
	}
	defer file.Close()

	// Write new content
	newContent := "new log entry\n"
	if _, err := file.WriteString(newContent); err != nil {
		t.Fatalf("Failed to write to log file: %v", err)
	}
	file.Close()

	// Read the file and verify both entries are present
	content, err := os.ReadFile(todayLogPath)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	expected := existingContent + newContent
	if string(content) != expected {
		t.Errorf("Log file content mismatch.\nExpected: %q\nGot: %q", expected, string(content))
	}
}

package cmd

import (
	"strings"
	"testing"
	"time"
)

func TestShowBanner(t *testing.T) {
	// Test that ShowBanner runs without panic
	// Detailed output checking is difficult due to ANSI codes and terminal formatting
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("ShowBanner panicked: %v", r)
		}
	}()

	ShowBanner()
	// If we got here without panic, the test passes
}

func TestShowSpinner(t *testing.T) {
	// Skip this test when running with race detector enabled
	// The race condition is in pterm's library code, not our code
	// pterm's SpinnerPrinter has unsynchronized access between Start() goroutine and Stop()
	// See: https://github.com/pterm/pterm/blob/master/spinner_printer.go
	if isRaceDetectorEnabled() {
		t.Skip("Skipping TestShowSpinner under race detector due to pterm library race condition")
	}

	spinner := ShowSpinner("Testing spinner...")

	if spinner == nil {
		t.Error("ShowSpinner returned nil")
		return
	}

	// Add a small delay to let the spinner goroutine initialize
	time.Sleep(50 * time.Millisecond)

	// Stop the spinner
	spinner.Stop()

	// Give time for cleanup
	time.Sleep(10 * time.Millisecond)
}

// isRaceDetectorEnabled detects if the binary was compiled with -race flag
// This is determined by checking if the race package is available
func isRaceDetectorEnabled() bool {
	// The Go runtime sets this when -race flag is used during build
	return raceDetectorEnabled
}

func TestShowSuccess(t *testing.T) {
	// Test that ShowSuccess runs without panic
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("ShowSuccess panicked: %v", r)
		}
	}()

	ShowSuccess("Test success message")
}

func TestShowError(t *testing.T) {
	// Test that ShowError runs without panic
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("ShowError panicked: %v", r)
		}
	}()

	ShowError("Test error message")
}

func TestShowInfo(t *testing.T) {
	// Test that ShowInfo runs without panic
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("ShowInfo panicked: %v", r)
		}
	}()

	ShowInfo("Test info message")
}

func TestShowWarning(t *testing.T) {
	// Test that ShowWarning runs without panic
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("ShowWarning panicked: %v", r)
		}
	}()

	ShowWarning("Test warning message")
}

func TestShowBox(t *testing.T) {
	// Test that ShowBox runs without panic
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("ShowBox panicked: %v", r)
		}
	}()

	ShowBox("Test Title", "Test content")
}

func TestShowProgress(t *testing.T) {
	progressbar := ShowProgress(100, "Testing progress...")

	if progressbar == nil {
		t.Error("ShowProgress returned nil")
	}

	// Stop the progress bar
	progressbar.Stop()
}

func TestVersionConstant(t *testing.T) {
	if Version == "" {
		t.Error("Version constant is empty")
	}

	// Version should follow semantic versioning pattern (x.y.z)
	parts := strings.Split(Version, ".")
	if len(parts) != 3 {
		t.Errorf("Version should be in format x.y.z, got: %s", Version)
	}
}

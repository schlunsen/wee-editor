package logging

import (
	"log"
	"os"
	"strings"
	"testing"
)

func TestStandardLogsPersistWithoutVerbose(t *testing.T) {
	l, err := newLogger(t.TempDir(), false)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		l.file.Close()
		l.errorFile.Close()
	})
	standard := log.New(l.StandardLogWriter(), "", 0)
	standard.Print("session diagnostic")
	standard.Print("ERROR: session failed")
	contents, err := os.ReadFile(l.GetLogFilePath())
	if err != nil {
		t.Fatal(err)
	}
	for _, message := range []string{"session diagnostic", "ERROR: session failed"} {
		if !strings.Contains(string(contents), message) {
			t.Errorf("missing standard log message %q", message)
		}
	}
}

package testutil

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func CaptureStdout(f func()) string {
	// Create a new read-write pipe
	r, w, _ := os.Pipe()
	// Redirect os.Stdout to the write end of the pipe
	os.Stdout = w

	// Use a separate goroutine to read from the pipe to prevent deadlocks
	// (reading and writing in the same goroutine can cause issues)
	var buf bytes.Buffer
	done := make(chan bool)
	go func() {
		_, err := io.Copy(&buf, r)
		if err != nil {
			panic(err)
		}
		done <- true
	}()

	f()

	// 5. Close the write end of the pipe to signal that writing is complete
	err := w.Close()
	if err != nil {
		panic(err)
	}
	<-done
	return buf.String()
}

func GoldenTest(t *testing.T, out string) {
	t.Helper()
	name := strings.ReplaceAll(t.Name(), "/", "_")
	name = strings.ReplaceAll(name, " ", "_")
	// Check for existing testdata to compare to
	wantPath := filepath.Join("testdata", name+".golden")
	wantPath, err := filepath.Abs(wantPath)
	if err != nil {
		t.Fatalf("abs golden output path: %v", err)
	}
	want, err := os.ReadFile(wantPath)
	if err != nil {
		want = []byte{}
	}
	// Compare the outputs
	if !bytes.Equal(want, []byte(out)) {
		// Write out to a file in /tmp
		gotFile, err := os.CreateTemp(os.TempDir(), name+"-*.got")
		if err != nil {
			t.Fatalf("create got output: %v", err)
		}
		gotPath := gotFile.Name()
		if _, err := gotFile.WriteString(out); err != nil {
			err = gotFile.Close()
			if err != nil {
				t.Error(err)
			}
			t.Fatalf("write got output: %v", err)
		}
		if err := gotFile.Close(); err != nil {
			t.Fatalf("close got output: %v", err)
		}
		// Display error
		t.Errorf("output mismatch\nwant: %s\ngot:  %s\ndiff: diff %s %s\ncopy: cp %s %s", wantPath, gotPath, gotPath, wantPath, gotPath, wantPath)
	}
}

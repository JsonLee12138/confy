package e2e_test

import (
	"bytes"
	"os"
	"testing"
)

// writeWatchedFile publishes a small, fixed-size fixture with one in-place write.
// Padding avoids a truncate notification exposing an intermediate empty document;
// keeping the inode avoids platform-specific Remove events from atomic replacement.
// This controls the test writer, not the production guarantees of file editors.
func writeWatchedFile(t *testing.T, path, content string) {
	t.Helper()
	const size = 4096
	if len(content) > size {
		t.Fatal("watched fixture exceeds fixed document size")
	}
	data := bytes.Repeat([]byte{' '}, size)
	copy(data, content)
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := file.WriteAt(data, 0); err != nil {
		_ = file.Close()
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
}

package confy

import (
	"os"
	"path/filepath"
	"testing"
)

func TestExport(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "config.yaml", "name: test\nport: 8080")

	cfg, err := New(WithPath(dir))
	if err != nil {
		t.Fatal(err)
	}

	outPath := filepath.Join(dir, "output", "exported.yaml")
	if err := cfg.Export(outPath); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(outPath); os.IsNotExist(err) {
		t.Error("expected exported file to exist")
	}
}

func TestExport_EmptyPath(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "config.yaml", "name: test")

	cfg, err := New(WithPath(dir))
	if err != nil {
		t.Fatal(err)
	}

	err = cfg.Export("")
	if err == nil {
		t.Error("expected error for empty path")
	}
}

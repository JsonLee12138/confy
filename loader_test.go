package confy

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/JsonLee12138/confy/internal"
)

func TestGetConfigFilePaths_Basic(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "config.yaml", "server:\n  port: 8080")

	opts := &configOptions{
		basePath: dir,
		fileName: "config",
		fileType: YAML,
	}

	paths := getConfigFilePaths(opts)
	if len(paths) != 1 {
		t.Fatalf("expected 1 path, got %d: %v", len(paths), paths)
	}
	if filepath.Base(paths[0]) != "config.yaml" {
		t.Errorf("expected config.yaml, got %s", filepath.Base(paths[0]))
	}
}

func TestGetConfigFilePaths_EnvOverride(t *testing.T) {
	dir := t.TempDir()
	internal.ResetMode()
	t.Setenv("GO_ENV_MODE", "production")

	writeFile(t, dir, "config.yaml", "server:\n  port: 8080")
	writeFile(t, dir, "config.production.yaml", "server:\n  port: 9090")

	opts := &configOptions{
		basePath: dir,
		fileName: "config",
		fileType: YAML,
	}

	paths := getConfigFilePaths(opts)
	if len(paths) != 2 {
		t.Fatalf("expected 2 paths, got %d: %v", len(paths), paths)
	}
}

func TestGetConfigFilePaths_LocalOverride(t *testing.T) {
	dir := t.TempDir()
	internal.ResetMode()
	t.Setenv("GO_ENV_MODE", "development")

	writeFile(t, dir, "config.yaml", "a: 1")
	writeFile(t, dir, "config.local.yaml", "a: 2")

	opts := &configOptions{
		basePath: dir,
		fileName: "config",
		fileType: YAML,
	}

	paths := getConfigFilePaths(opts)
	if len(paths) != 2 {
		t.Fatalf("expected 2 paths, got %d: %v", len(paths), paths)
	}
}

func TestGetConfigFilePaths_NoFiles(t *testing.T) {
	dir := t.TempDir()

	opts := &configOptions{
		basePath: dir,
		fileName: "config",
		fileType: YAML,
	}

	paths := getConfigFilePaths(opts)
	if len(paths) != 0 {
		t.Errorf("expected 0 paths, got %d", len(paths))
	}
}

func TestGetAllConfigFilePaths(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "config.yaml", "a: 1")
	writeFile(t, dir, "database.yaml", "host: localhost")
	writeFile(t, dir, "cache.yaml", "ttl: 60")

	opts := &configOptions{
		basePath: dir,
		fileName: "config",
		fileType: YAML,
		loadAll:  true,
	}

	paths := getAllConfigFilePaths(opts)
	if len(paths) < 3 {
		t.Fatalf("expected at least 3 paths, got %d: %v", len(paths), paths)
	}
	if filepath.Base(paths[0]) != "config.yaml" {
		t.Errorf("expected config.yaml first, got %s", filepath.Base(paths[0]))
	}
}

func TestMultiFormatDetection(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "config.yaml", "a: 1")
	writeFile(t, dir, "database.json", `{"host":"localhost"}`)

	opts := &configOptions{
		basePath: dir,
		fileName: "config",
		fileType: YAML,
		loadAll:  true,
	}

	paths := getAllConfigFilePaths(opts)
	if len(paths) < 2 {
		t.Fatalf("expected at least 2 paths, got %d: %v", len(paths), paths)
	}
}

func writeFile(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

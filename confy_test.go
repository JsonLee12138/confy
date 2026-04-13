package confy

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNew_Defaults(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "config.yaml", "server:\n  port: 8080")

	cfg, err := New(WithPath(dir))
	if err != nil {
		t.Fatal(err)
	}
	if cfg == nil {
		t.Fatal("expected non-nil config")
	}

	val := cfg.Get("server.port")
	if val == nil {
		t.Error("expected server.port to be set")
	}
}

func TestNew_NoConfigFiles(t *testing.T) {
	dir := t.TempDir()

	_, err := New(WithPath(dir))
	if err == nil {
		t.Error("expected error when no config files found")
	}
}

func TestNew_GetSet(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "config.yaml", "server:\n  port: 8080")

	cfg, err := New(WithPath(dir))
	if err != nil {
		t.Fatal(err)
	}

	val := cfg.Get("server.port")
	if val == nil {
		t.Fatal("expected value")
	}

	cfg.Set("server.port", 9090)
	val = cfg.Get("server.port")
	if val == nil {
		t.Fatal("expected value after set")
	}
}

func TestNew_YAML(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "config.yaml", "name: confy")

	cfg, err := New(WithPath(dir))
	if err != nil {
		t.Fatal(err)
	}

	if cfg.Get("name") != "confy" {
		t.Errorf("expected 'confy', got '%v'", cfg.Get("name"))
	}
}

func TestNew_JSON(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "config.json"), []byte(`{"name":"confy"}`), 0644)

	cfg, err := New(WithPath(dir), WithFileType(JSON))
	if err != nil {
		t.Fatal(err)
	}

	if cfg.Get("name") != "confy" {
		t.Errorf("expected 'confy', got '%v'", cfg.Get("name"))
	}
}

func TestNew_TOML(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "config.toml"), []byte("name = \"confy\"\n"), 0644)

	cfg, err := New(WithPath(dir), WithFileType(TOML))
	if err != nil {
		t.Fatal(err)
	}

	if cfg.Get("name") != "confy" {
		t.Errorf("expected 'confy', got '%v'", cfg.Get("name"))
	}
}

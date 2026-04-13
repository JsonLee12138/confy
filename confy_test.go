package confy

import (
	"fmt"
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

func TestNew_WithInheritance(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "base.yaml", "server:\n  port: 8080\n  host: 0.0.0.0")
	writeFile(t, dir, "config.yaml", "base: base.yaml\nserver:\n  port: 9090")

	cfg, err := New(WithPath(dir))
	if err != nil {
		t.Fatal(err)
	}

	if cfg.Get("server.port") != 9090 {
		t.Errorf("expected port 9090, got %v", cfg.Get("server.port"))
	}
	if cfg.Get("server.host") != "0.0.0.0" {
		t.Errorf("expected host '0.0.0.0', got %v", cfg.Get("server.host"))
	}
}

func TestNew_WithEncryption(t *testing.T) {
	dir := t.TempDir()
	key := make([]byte, 32)
	copy(key, []byte("0123456789abcdef0123456789abcdef"))

	encVal, _ := encryptConfigValue("secret_password", key)
	writeFile(t, dir, "config.yaml", fmt.Sprintf("database:\n  password: %s\n  host: localhost", encVal))

	cfg, err := New(
		WithPath(dir),
		WithEncryption("aes-gcm", key),
	)
	if err != nil {
		t.Fatal(err)
	}

	var appCfg struct {
		Database struct {
			Password string `mapstructure:"password"`
			Host     string `mapstructure:"host"`
		} `mapstructure:"database"`
	}

	if err := cfg.Bind(&appCfg); err != nil {
		t.Fatal(err)
	}

	if appCfg.Database.Password != "secret_password" {
		t.Errorf("expected decrypted 'secret_password', got '%s'", appCfg.Database.Password)
	}
	if appCfg.Database.Host != "localhost" {
		t.Errorf("expected 'localhost', got '%s'", appCfg.Database.Host)
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

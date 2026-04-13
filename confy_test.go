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

func TestEndToEnd(t *testing.T) {
	dir := t.TempDir()

	// Base config
	writeFile(t, dir, "base.yaml", `
server:
  host: 0.0.0.0
  port: 8080
database:
  host: localhost
  port: 5432
`)

	// Main config with inheritance + encrypted value
	key := make([]byte, 32)
	copy(key, []byte("0123456789abcdef0123456789abcdef"))
	encPass, _ := encryptConfigValue("my_secret", key)

	writeFile(t, dir, "config.yaml", fmt.Sprintf(`
base: base.yaml
server:
  port: 9090
database:
  password: %s
`, encPass))

	// Local override
	writeFile(t, dir, "config.local.yaml", `
server:
  host: 127.0.0.1
`)

	// .env file
	writeFile(t, dir, ".env", "APP_NAME=confy-test\n")

	cfg, err := New(
		WithPath(dir),
		WithEnvPrefix("TEST"),
		WithEncryption("aes-gcm", key),
		WithDotEnv(filepath.Join(dir, ".env")),
	)
	if err != nil {
		t.Fatal(err)
	}

	var appCfg struct {
		Server struct {
			Host string `mapstructure:"host"`
			Port int    `mapstructure:"port"`
		} `mapstructure:"server"`
		Database struct {
			Host     string `mapstructure:"host"`
			Port     int    `mapstructure:"port"`
			Password string `mapstructure:"password"`
		} `mapstructure:"database"`
	}

	if err := cfg.Bind(&appCfg); err != nil {
		t.Fatal(err)
	}

	// Inheritance: base port overridden by config
	if appCfg.Server.Port != 9090 {
		t.Errorf("expected port 9090, got %d", appCfg.Server.Port)
	}

	// Local override: host overridden
	if appCfg.Server.Host != "127.0.0.1" {
		t.Errorf("expected host '127.0.0.1', got '%s'", appCfg.Server.Host)
	}

	// Inheritance: database from base
	if appCfg.Database.Host != "localhost" {
		t.Errorf("expected db host 'localhost', got '%s'", appCfg.Database.Host)
	}
	if appCfg.Database.Port != 5432 {
		t.Errorf("expected db port 5432, got %d", appCfg.Database.Port)
	}

	// Decryption
	if appCfg.Database.Password != "my_secret" {
		t.Errorf("expected decrypted 'my_secret', got '%s'", appCfg.Database.Password)
	}

	// .env loaded
	if os.Getenv("APP_NAME") != "confy-test" {
		t.Errorf("expected APP_NAME 'confy-test', got '%s'", os.Getenv("APP_NAME"))
	}

	// Snapshot/Restore
	snap, err := cfg.Snapshot()
	if err != nil {
		t.Fatal(err)
	}
	cfg.Set("server.port", 1111)
	if err := cfg.Restore(snap); err != nil {
		t.Fatal(err)
	}

	// Export
	outPath := filepath.Join(dir, "output", "exported.yaml")
	if err := cfg.Export(outPath); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(outPath); os.IsNotExist(err) {
		t.Error("exported file should exist")
	}
}

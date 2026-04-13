package confy

import (
	"testing"
)

type TestAppConfig struct {
	Server struct {
		Port int    `mapstructure:"port"`
		Host string `mapstructure:"host"`
	} `mapstructure:"server"`
}

type TestDefaultConfig struct {
	Name    string `mapstructure:"name" default:"confy"`
	Timeout int    `mapstructure:"timeout" default:"30"`
}

func TestBind(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "config.yaml", "server:\n  port: 8080\n  host: localhost")

	cfg, err := New(WithPath(dir))
	if err != nil {
		t.Fatal(err)
	}

	var appCfg TestAppConfig
	if err := cfg.Bind(&appCfg); err != nil {
		t.Fatal(err)
	}

	if appCfg.Server.Port != 8080 {
		t.Errorf("expected port 8080, got %d", appCfg.Server.Port)
	}
	if appCfg.Server.Host != "localhost" {
		t.Errorf("expected host 'localhost', got '%s'", appCfg.Server.Host)
	}
}

func TestBind_NilConfig(t *testing.T) {
	var cfg *Config
	var appCfg TestAppConfig

	err := cfg.Bind(&appCfg)
	if err == nil {
		t.Error("expected error for nil config")
	}
}

func TestBind_NilTarget(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "config.yaml", "name: test")

	cfg, err := New(WithPath(dir))
	if err != nil {
		t.Fatal(err)
	}

	err = cfg.Bind(nil)
	if err == nil {
		t.Error("expected error for nil target")
	}
}

func TestBindWithDefaults(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "config.yaml", "name: custom")

	cfg, err := New(WithPath(dir))
	if err != nil {
		t.Fatal(err)
	}

	var appCfg TestDefaultConfig
	if err := cfg.BindWithDefaults(&appCfg); err != nil {
		t.Fatal(err)
	}

	if appCfg.Name != "custom" {
		t.Errorf("expected name 'custom', got '%s'", appCfg.Name)
	}
	if appCfg.Timeout != 30 {
		t.Errorf("expected timeout 30 (default), got %d", appCfg.Timeout)
	}
}

func TestBind_WatchRebinds(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "config.yaml", "server:\n  port: 8080")

	cfg, err := New(WithPath(dir), WithWatch(true))
	if err != nil {
		t.Fatal(err)
	}

	var appCfg TestAppConfig
	if err := cfg.Bind(&appCfg); err != nil {
		t.Fatal(err)
	}

	if appCfg.Server.Port != 8080 {
		t.Errorf("expected port 8080, got %d", appCfg.Server.Port)
	}
}

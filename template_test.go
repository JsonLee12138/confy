package confy

import (
	"path/filepath"
	"testing"
)

func TestResolveBaseInheritance(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "base.yaml", "server:\n  port: 8080\n  host: 0.0.0.0\ndatabase:\n  host: localhost")
	writeFile(t, dir, "config.yaml", "base: base.yaml\nserver:\n  port: 9090")

	resolved, err := resolveBaseInheritance(filepath.Join(dir, "config.yaml"), YAML, make(map[string]bool))
	if err != nil {
		t.Fatal(err)
	}

	if resolved["server.port"] != 9090 {
		t.Errorf("expected server.port 9090, got %v", resolved["server.port"])
	}
	if resolved["server.host"] != "0.0.0.0" {
		t.Errorf("expected server.host '0.0.0.0', got %v", resolved["server.host"])
	}
	if resolved["database.host"] != "localhost" {
		t.Errorf("expected database.host 'localhost', got %v", resolved["database.host"])
	}
}

func TestResolveBaseInheritance_Chain(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "root.yaml", "name: root\nlevel: 1")
	writeFile(t, dir, "middle.yaml", "base: root.yaml\nlevel: 2")
	writeFile(t, dir, "config.yaml", "base: middle.yaml\nlevel: 3")

	resolved, err := resolveBaseInheritance(filepath.Join(dir, "config.yaml"), YAML, make(map[string]bool))
	if err != nil {
		t.Fatal(err)
	}

	if resolved["name"] != "root" {
		t.Errorf("expected name 'root', got %v", resolved["name"])
	}
	if resolved["level"] != 3 {
		t.Errorf("expected level 3, got %v", resolved["level"])
	}
}

func TestResolveBaseInheritance_Cycle(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "a.yaml", "base: b.yaml")
	writeFile(t, dir, "b.yaml", "base: a.yaml")

	_, err := resolveBaseInheritance(filepath.Join(dir, "a.yaml"), YAML, make(map[string]bool))
	if err == nil {
		t.Error("expected cycle detection error")
	}
}

func TestResolveBaseInheritance_NoBase(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "config.yaml", "name: test\nport: 8080")

	resolved, err := resolveBaseInheritance(filepath.Join(dir, "config.yaml"), YAML, make(map[string]bool))
	if err != nil {
		t.Fatal(err)
	}

	if resolved["name"] != "test" {
		t.Errorf("expected name 'test', got %v", resolved["name"])
	}
}

func TestMergeConfig(t *testing.T) {
	base := map[string]any{
		"server.port": 8080,
		"server.host": "0.0.0.0",
		"name":        "base",
	}
	override := map[string]any{
		"server.port": 9090,
		"name":        "override",
	}

	merged := mergeConfig(base, override)

	if merged["server.port"] != 9090 {
		t.Errorf("expected port 9090, got %v", merged["server.port"])
	}
	if merged["server.host"] != "0.0.0.0" {
		t.Errorf("expected host '0.0.0.0', got %v", merged["server.host"])
	}
	if merged["name"] != "override" {
		t.Errorf("expected name 'override', got %v", merged["name"])
	}
}

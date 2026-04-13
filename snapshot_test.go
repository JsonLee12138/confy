package confy

import (
	"testing"
)

func TestSnapshot(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "config.yaml", "name: test\nport: 8080")

	cfg, err := New(WithPath(dir))
	if err != nil {
		t.Fatal(err)
	}

	snap, err := cfg.Snapshot()
	if err != nil {
		t.Fatal(err)
	}
	if snap == nil {
		t.Fatal("expected non-nil snapshot")
	}
}

func TestSnapshot_Restore(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "config.yaml", "name: test\nport: 8080")

	cfg, err := New(WithPath(dir))
	if err != nil {
		t.Fatal(err)
	}

	snap, err := cfg.Snapshot()
	if err != nil {
		t.Fatal(err)
	}

	cfg.Set("name", "changed")

	if err := cfg.Restore(snap); err != nil {
		t.Fatal(err)
	}
}

func TestRestore_NilSnapshot(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "config.yaml", "name: test")

	cfg, err := New(WithPath(dir))
	if err != nil {
		t.Fatal(err)
	}

	err = cfg.Restore(nil)
	if err == nil {
		t.Error("expected error for nil snapshot")
	}
}

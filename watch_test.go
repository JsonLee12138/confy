package confy

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestWatch_ConfigChange(t *testing.T) {
	dir := t.TempDir()
	configFile := filepath.Join(dir, "config.yaml")
	writeFile(t, dir, "config.yaml", "name: original")

	changed := make(chan struct{}, 1)

	cfg, err := New(
		WithPath(dir),
		WithWatch(true),
		WithOnChange(func(e Event) {
			select {
			case changed <- struct{}{}:
			default:
			}
		}),
	)
	if err != nil {
		t.Fatal(err)
	}

	var appCfg struct {
		Name string `mapstructure:"name"`
	}
	if err := cfg.Bind(&appCfg); err != nil {
		t.Fatal(err)
	}

	// Modify the file
	time.Sleep(100 * time.Millisecond) // let watcher initialize
	os.WriteFile(configFile, []byte("name: updated"), 0644)

	select {
	case <-changed:
		// Success - config was reloaded
	case <-time.After(5 * time.Second):
		t.Error("config change callback was not triggered within timeout")
	}
}

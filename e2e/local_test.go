package e2e_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/JsonLee12138/confy"
	"github.com/JsonLee12138/confy/provider/local"
)

// TestE2ELocalProvider exercises real file reloads, inheritance, ENV/Set precedence,
// failed-document preservation, recovery, and closed-Config behavior.
func TestE2ELocalProvider(t *testing.T) {
	dir := t.TempDir()
	write := func(name, content string) {
		t.Helper()
		// Exercise native in-place file writes. Atomic replacement may produce a
		// Remove event on macOS, which ends Viper's native watcher.
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0600); err != nil {
			t.Fatal(err)
		}
	}
	write("base.yaml", "server:\n  port: 8080\n")
	write("config.yaml", "base: base.yaml\nname: initial\nremoved: old\ntimeout: 0\nenabled: false\n")
	t.Setenv("CONFY_E2E_SERVER_PORT", "9090")
	p := local.New(local.WithPath(dir), local.WithLoadAll(false))
	var target e2eSettings
	updates := make(chan e2eSettings, 32)
	failures := make(chan error, 32)
	cfg, err := confy.New(confy.WithProvider(p), confy.WithEnvPrefix("CONFY_E2E"), confy.WithWatch(true),
		confy.WithOnChange(func(confy.Event) { updates <- target }),
		confy.WithOnError(func(err error) { failures <- err }))
	if err != nil {
		t.Fatal(err)
	}
	defer cfg.Close()
	if err := cfg.Bind(&target); err != nil {
		t.Fatal(err)
	}
	if target.Name != "initial" || target.Server.Port != 9090 || target.Timeout != 0 || target.Enabled {
		t.Fatalf("initial binding = %+v", target)
	}
	cfg.Set("server.port", 10001)
	wait := func(name string) e2eSettings {
		t.Helper()
		timer := time.NewTimer(10 * time.Second)
		defer timer.Stop()
		for {
			select {
			case got := <-updates:
				if got.Name == name {
					return got
				}
			case <-timer.C:
				t.Fatalf("no local update for %s", name)
				return e2eSettings{}
			}
		}
	}
	write("config.yaml", "base: base.yaml\nname: updated\n")
	got := wait("updated")
	if got.Removed != "" || got.Timeout != 30 || !got.Enabled || got.Server.Port != 10001 {
		t.Fatalf("updated binding = %+v", got)
	}
	write("config.yaml", "name: [broken")
	select {
	case <-failures:
	case <-time.After(10 * time.Second):
		t.Fatal("invalid local update was not reported")
	}
	if cfg.Get("name") != "updated" {
		t.Fatal("invalid document changed Viper")
	}
	p.ConfigMutex().RLock()
	unchanged := target.Name == "updated"
	p.ConfigMutex().RUnlock()
	if !unchanged {
		t.Fatal("invalid document changed target")
	}
	write("config.yaml", "base: base.yaml\nname: recovered\n")
	if got := wait("recovered"); got.Server.Port != 10001 {
		t.Fatal("Set lost after recovery")
	}
	if err := cfg.Close(); err != nil {
		t.Fatal(err)
	}
	if err := cfg.Bind(&target); !errors.Is(err, confy.ErrClosed) {
		t.Fatalf("bind after Close = %v", err)
	}
	t.Log("real files: inheritance, ENV/Set precedence, zero/default/delete, watch, invalid document, recovery, Close passed")
}

// TestE2ELocalMixedFormatWatch checks mixed YAML/TOML inheritance during initial binding
// and automatic rebinds, including a TOML file discovered with the default YAML preference.
func TestE2ELocalMixedFormatWatch(t *testing.T) {
	for _, tc := range []struct {
		name, parentName, parent, childName, child, updated string
		format                                              confy.FileType
	}{
		{"yaml-from-toml", "base.toml", "port = 8080\n", "config.yaml", "base: base.toml\nname: initial\n", "base: base.toml\nname: updated\n", confy.YAML},
		{"toml-with-yaml-default", "base.yaml", "port: 8080\n", "config.toml", "base = \"base.yaml\"\nname = \"initial\"\n", "base = \"base.yaml\"\nname = \"updated\"\n", confy.YAML},
		{"toml-from-yaml", "base.yaml", "port: 8080\n", "config.toml", "base = \"base.yaml\"\nname = \"initial\"\n", "base = \"base.yaml\"\nname = \"updated\"\n", confy.TOML},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			write := func(name, content string) {
				t.Helper()
				if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0600); err != nil {
					t.Fatal(err)
				}
			}
			write(tc.parentName, tc.parent)
			write(tc.childName, tc.child)
			var target struct {
				Name string
				Port int
			}
			updates := make(chan struct {
				Name string
				Port int
			}, 16)
			cfg, err := confy.New(confy.WithProvider(local.New(local.WithPath(dir), local.WithFileType(tc.format), local.WithLoadAll(false))), confy.WithWatch(true), confy.WithOnChange(func(confy.Event) { updates <- target }))
			if err != nil {
				t.Fatal(err)
			}
			defer cfg.Close()
			if err := cfg.Bind(&target); err != nil {
				t.Fatal(err)
			}
			if target.Name != "initial" || target.Port != 8080 {
				t.Fatalf("initial = %+v", target)
			}
			write(tc.childName, tc.updated)
			timer := time.NewTimer(10 * time.Second)
			defer timer.Stop()
			for {
				select {
				case got := <-updates:
					if got.Name == "updated" {
						if got.Port != 8080 {
							t.Fatal("inherited value lost")
						}
						return
					}
				case <-timer.C:
					t.Fatal("mixed-format watch did not rebind")
					return
				}
			}
		})
	}
}

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

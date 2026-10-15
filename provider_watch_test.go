package confy

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/spf13/viper"
)

type queuedProvider struct {
	staticProvider
	events chan Change
	done   chan struct{}
	once   sync.Once
	starts int
	fail   bool
	closed bool
}

func (p *queuedProvider) Watch(context.Context) (Watcher, error) {
	p.starts++
	if p.fail {
		p.fail = false
		return nil, errors.New("registration")
	}
	return p, nil
}
func (p *queuedProvider) Next(ctx context.Context) (Change, error) {
	select {
	case <-ctx.Done():
		return Change{}, ctx.Err()
	case <-p.done:
		return Change{}, io.EOF
	case c := <-p.events:
		return c, nil
	}
}
func (p *queuedProvider) Close() error {
	p.once.Do(func() { p.closed = true; close(p.done) })
	return nil
}
func valueConfig(values map[string]any) *viper.Viper {
	v := viper.New()
	for k, x := range values {
		v.Set(k, x)
	}
	return v
}
func queueProvider() *queuedProvider {
	return &queuedProvider{staticProvider: staticProvider{value: valueConfig(map[string]any{"name": "old", "removed": "old", "port": 1, "timeout": 0, "secret": ""})}, events: make(chan Change, 16), done: make(chan struct{})}
}

type reloadTarget struct {
	Name    string
	Removed string
	Port    int
	Timeout int `default:"30"`
	Secret  string
}

func (v *reloadTarget) Validate() error {
	if v.Port < 0 {
		return errors.New("private validation content")
	}
	return nil
}
func receive[T any](t *testing.T, ch <-chan T) T {
	t.Helper()
	select {
	case v := <-ch:
		return v
	case <-time.After(5 * time.Second):
		t.Fatal("notification timeout")
		var zero T
		return zero
	}
}
func TestProviderReloadTransaction(t *testing.T) {
	p := queueProvider()
	updated := make(chan reloadTarget, 16)
	failed := make(chan error, 16)
	t.Setenv("CONFYTEST_SECRET", "env-secret")
	var target reloadTarget
	cfg, err := New(WithProvider(p), WithEnvPrefix("CONFYTEST"), WithWatch(true), WithOnChange(func(Event) { updated <- target }), WithOnError(func(err error) { failed <- err }))
	if err != nil {
		t.Fatal(err)
	}
	defer cfg.Close()
	if err = cfg.BindWithDefaults(&target); err != nil {
		t.Fatal(err)
	}
	cfg.Set("name", "explicit")
	send := func(values map[string]any) { p.events <- Change{Config: valueConfig(values)} }
	send(map[string]any{"name": "new", "port": 2, "secret": "remote"})
	got := receive(t, updated)
	if got.Name != "explicit" || got.Removed != "" || got.Timeout != 30 || got.Secret != "env-secret" || got.Port != 2 {
		t.Fatalf("candidate: %+v", got)
	}
	if cfg.Get("removed") != nil {
		t.Fatal("deleted value retained")
	}
	for _, bad := range []map[string]any{{"port": -1}, {"port": "not-an-int"}, {"removed": "enc:AES_GCM:broken"}} {
		send(bad)
		if err := receive(t, failed); !errors.Is(err, ErrCandidate) {
			t.Fatal(err)
		}
		if cfg.Get("port") != 2 {
			t.Fatal("invalid candidate committed")
		}
	}
	send(map[string]any{"port": 3})
	if got = receive(t, updated); got.Port != 3 {
		t.Fatal(got)
	}
	if err = cfg.Bind(&reloadTarget{}); !errors.Is(err, ErrBoundTarget) {
		t.Fatal(err)
	}
	if err = cfg.Close(); err != nil {
		t.Fatal(err)
	}
	cfg.Set("name", "after-close")
	if cfg.Get("name") != "explicit" {
		t.Fatal("closed Set mutated config")
	}
	if err = cfg.Bind(&target); !errors.Is(err, ErrClosed) {
		t.Fatal(err)
	}
	if err = cfg.Restore(map[string]any{"name": "x"}); !errors.Is(err, ErrClosed) {
		t.Fatal(err)
	}
}
func TestWatchRegistrationRetryAndCallbackReentry(t *testing.T) {
	p := queueProvider()
	p.fail = true
	done := make(chan error, 1)
	var target reloadTarget
	var cfg *Config
	cfg, err := New(WithProvider(p), WithWatch(true), WithOnChange(func(Event) {
		cfg.Get("name")
		cfg.Set("callback", true)
		err := cfg.Bind(&target)
		if err == nil {
			err = cfg.Close()
		}
		done <- err
	}))
	if err != nil {
		t.Fatal(err)
	}
	defer cfg.Close()
	if err = cfg.Bind(&target); err == nil {
		t.Fatal("registration failure lost")
	}
	if err = cfg.Bind(&target); err != nil {
		t.Fatal(err)
	}
	if err = cfg.Bind(&target); err != nil {
		t.Fatal(err)
	}
	if p.starts != 2 {
		t.Fatal(p.starts)
	}
	p.events <- Change{Config: valueConfig(map[string]any{"name": "next"})}
	if err = receive(t, done); err != nil {
		t.Fatal(err)
	}
}
func TestWatchUnsupportedAndFailedLoadCleanup(t *testing.T) {
	p := &staticProvider{value: valueConfig(map[string]any{"name": "x"})}
	c, err := New(WithProvider(p), WithWatch(true))
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	if err = c.Bind(&reloadTarget{}); !errors.Is(err, ErrWatchUnsupported) {
		t.Fatal(err)
	}
	q := queueProvider()
	q.err = errors.New("load")
	if _, err = New(WithProvider(q)); err == nil || !q.closed {
		t.Fatal("failed load not cleaned")
	}
}
func TestProviderConcurrentAccess(t *testing.T) {
	p := queueProvider()
	changed := make(chan struct{}, 64)
	c, err := New(WithProvider(p), WithWatch(true), WithOnChange(func(Event) { changed <- struct{}{} }))
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	var target reloadTarget
	if err = c.Bind(&target); err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				c.Set("override", j)
				c.Get("port")
				_, _ = c.Snapshot()
			}
		}()
	}
	for i := 0; i < 10; i++ {
		p.events <- Change{Config: valueConfig(map[string]any{"port": i})}
		receive(t, changed)
	}
	wg.Wait()
	c.Set("override", 999)
	p.events <- Change{Config: valueConfig(map[string]any{"port": 11})}
	receive(t, changed)
	if c.Get("override") != 999 {
		t.Fatal("override lost")
	}
}
func TestLocalAtomicReplaceAndFirstTarget(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "config.yaml", "name: old")
	changed := make(chan string, 16)
	var first, second struct{ Name string }
	c, err := New(WithPath(dir), WithWatch(true), WithOnChange(func(Event) { changed <- first.Name }))
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	if err = c.Bind(&first); err != nil {
		t.Fatal(err)
	}
	if err = c.Bind(&second); err != nil {
		t.Fatal(err)
	}
	temp := filepath.Join(dir, "replace.tmp")
	if err = os.WriteFile(temp, []byte("name: replaced"), 0600); err != nil {
		t.Fatal(err)
	}
	if err = os.Rename(temp, filepath.Join(dir, "config.yaml")); err != nil {
		t.Fatal(err)
	}
	if name := receive(t, changed); name != "replaced" {
		t.Fatal(name)
	}
	if second.Name != "old" {
		t.Fatal("historical second target changed")
	}
}

func TestRemoteEncryptionRollback(t *testing.T) {
	key := []byte("01234567890123456789012345678901")
	encrypted, err := encryptConfigValue("initial-secret", key)
	if err != nil {
		t.Fatal(err)
	}
	for _, opts := range [][]Option{nil, {WithEncryption("aes-gcm", []byte("bad"))}} {
		p := &staticProvider{value: valueConfig(map[string]any{"secret": encrypted})}
		cfg, err := New(append(opts, WithProvider(p))...)
		if err != nil {
			t.Fatal(err)
		}
		if err = cfg.Bind(&reloadTarget{}); !errors.Is(err, ErrCandidate) {
			t.Fatalf("missing/wrong key: %v", err)
		}
		_ = cfg.Close()
	}
	p := queueProvider()
	p.value = valueConfig(map[string]any{"secret": encrypted, "port": 1})
	updates := make(chan reloadTarget, 8)
	rejected := make(chan error, 8)
	var target reloadTarget
	cfg, err := New(WithProvider(p), WithEncryption("aes-gcm", key), WithWatch(true), WithOnChange(func(Event) { updates <- target }), WithOnError(func(err error) { rejected <- err }))
	if err != nil {
		t.Fatal(err)
	}
	defer cfg.Close()
	if err = cfg.Bind(&target); err != nil || target.Secret != "initial-secret" {
		t.Fatal(err)
	}
	p.events <- Change{Config: valueConfig(map[string]any{"secret": "enc:AES_GCM:broken", "port": 2})}
	receive(t, rejected)
	if cfg.Get("port") != 1 {
		t.Fatal("decryption failure committed")
	}
	next, err := encryptConfigValue("next-secret", key)
	if err != nil {
		t.Fatal(err)
	}
	p.events <- Change{Config: valueConfig(map[string]any{"secret": next, "port": 3})}
	if got := receive(t, updates); got.Secret != "next-secret" {
		t.Fatal("encrypted update not decoded")
	}
}

func TestReloadDedupIncludesOverrides(t *testing.T) {
	p := queueProvider()
	updated := make(chan string, 8)
	var target reloadTarget
	c, err := New(WithProvider(p), WithWatch(true), WithOnChange(func(Event) { updated <- target.Name }))
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	if err = c.Bind(&target); err != nil {
		t.Fatal(err)
	}
	same := func() { p.events <- Change{Config: valueConfig(map[string]any{"name": "same"})} }
	same()
	receive(t, updated)
	c.Set("name", "override")
	same()
	if got := receive(t, updated); got != "override" {
		t.Fatal(got)
	}
	// A subsequent distinct update acts as a barrier: a duplicate must not notify.
	same()
	p.events <- Change{Config: valueConfig(map[string]any{"name": "next", "port": 2})}
	receive(t, updated)
	select {
	case <-updated:
		t.Fatal("duplicate notified")
	default:
	}
}

func TestLocalWatchReloadPreservesMergePriority(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "config.yaml", "name: base\nport: 1")
	writeFile(t, dir, "config.local.yaml", "port: 42")
	changed := make(chan reloadTarget, 8)
	var target reloadTarget
	c, err := New(WithPath(dir), WithWatch(true), WithOnChange(func(Event) { changed <- target }))
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	if err = c.Bind(&target); err != nil {
		t.Fatal(err)
	}
	if target.Port != 42 {
		t.Fatal("initial precedence changed")
	}
	if err = os.WriteFile(filepath.Join(dir, "config.yaml"), []byte("name: changed\nport: 2"), 0600); err != nil {
		t.Fatal(err)
	}
	got := receive(t, changed)
	if got.Port != 42 || got.Name != "changed" {
		t.Fatalf("reload lost priority: %+v", got)
	}
}

func TestLocalAndRemoteBindEquivalence(t *testing.T) {
	t.Setenv("CONFYEQ_SECRET", "from-env")
	formats := map[FileType]string{YAML: "name: same\nport: 12\nsecret: ''", JSON: `{"name":"same","port":12,"secret":""}`, TOML: "name = 'same'\nport = 12\nsecret = ''"}
	for format, content := range formats {
		t.Run(string(format), func(t *testing.T) {
			dir := t.TempDir()
			writeFile(t, dir, "config."+string(format), content)
			local, err := New(WithPath(dir), WithFileType(format), WithEnvPrefix("CONFYEQ"))
			if err != nil {
				t.Fatal(err)
			}
			defer local.Close()
			v := viper.New()
			v.SetConfigType(string(format))
			if err = v.ReadConfig(strings.NewReader(content)); err != nil {
				t.Fatal(err)
			}
			p := &staticProvider{value: v}
			remote, err := New(WithProvider(p), WithEnvPrefix("CONFYEQ"))
			if err != nil {
				t.Fatal(err)
			}
			defer remote.Close()
			var left, right reloadTarget
			if err = local.BindWithDefaults(&left); err != nil {
				t.Fatal(err)
			}
			if err = remote.BindWithDefaults(&right); err != nil {
				t.Fatal(err)
			}
			if left != right {
				t.Fatalf("local=%+v remote=%+v", left, right)
			}
		})
	}
}
func TestDisabledWatchNeverSubscribes(t *testing.T) {
	p := queueProvider()
	cfg, err := New(WithProvider(p), WithWatch(false), WithOnChange(func(Event) { t.Error("disabled watcher notified") }))
	if err != nil {
		t.Fatal(err)
	}
	defer cfg.Close()
	var target reloadTarget
	if err = cfg.Bind(&target); err != nil {
		t.Fatal(err)
	}
	if p.starts != 0 {
		t.Fatal("disabled Watch called provider")
	}
}

package nacos

import (
	"context"
	"errors"
	"github.com/JsonLee12138/confy/provider"
	"github.com/nacos-group/nacos-sdk-go/v2/common/constant"
	"github.com/nacos-group/nacos-sdk-go/v2/vo"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/JsonLee12138/confy"
)

// TestInvalidOptionsAndDocuments rejects unsafe source settings and invalid or empty documents.
func TestInvalidOptionsAndDocuments(t *testing.T) {
	for _, opts := range [][]Option{nil, {WithDataID("x")}, {WithDataID("x"), WithFormat("xml")}} {
		p := New(opts...)
		if _, err := p.Load(context.Background()); !errors.Is(err, ErrOptions) {
			t.Fatalf("expected options error: %v", err)
		}
		_ = p.Close()
	}
	for _, format := range []confy.FileType{confy.YAML, confy.JSON, confy.TOML} {
		for _, content := range []string{"", "  ", "[]", "null", "{}"} {
			p := New(WithFormat(format))
			if err := p.writeConfig(context.Background(), content); err == nil {
				t.Fatalf("accepted %s %q", format, content)
			}
		}
	}
	for format, content := range map[confy.FileType]string{confy.YAML: "name: test", confy.JSON: `{"name":"test"}`, confy.TOML: `name = "test"`} {
		p := New(WithFormat(format))
		err := p.writeConfig(context.Background(), content)
		if err != nil || p.v.GetString("name") != "test" {
			t.Fatalf("%s: %v", format, err)
		}
	}
}

type fakeSDK struct {
	mu                      sync.Mutex
	content                 string
	readErr, listenErr      error
	callback                vo.Listener
	reads, canceled, closed int
	observed                chan string
}

// GetConfig returns fake source state under a mutex and optionally signals a read barrier.
func (f *fakeSDK) GetConfig(vo.ConfigParam) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.reads++
	if f.observed != nil {
		select {
		case f.observed <- f.content:
		default:
		}
	}
	return f.content, f.readErr
}

// ListenConfig records the subscriber callback and returns the configured registration failure.
func (f *fakeSDK) ListenConfig(p vo.ConfigParam) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.callback = p.OnChange
	return f.listenErr
}

// CancelListenConfig counts cancellation attempts without contacting a server.
func (f *fakeSDK) CancelListenConfig(vo.ConfigParam) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.canceled++
	return nil
}

// CloseClient records ownership cleanup for lifecycle assertions.
func (f *fakeSDK) CloseClient() {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.closed++
}

// setContent replaces fake remote content without notifying the subscriber.
func (f *fakeSDK) setContent(content string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.content = content
}

// notify dispatches deliberately stale callback content to verify the watcher rereads the source.
func (f *fakeSDK) notify() {
	f.mu.Lock()
	callback := f.callback
	f.mu.Unlock()
	callback("", "", "", "out-of-order-old-content")
}

// watchEvents subscribes with buffered result channels and registers watcher cleanup.
func watchEvents(t *testing.T, p *Provider, ctx context.Context) (provider.Watcher, <-chan provider.Change, <-chan error) {
	t.Helper()
	changes := make(chan provider.Change, 2048)
	failures := make(chan error, 16)
	w, err := p.Watch(ctx, func(c provider.Change) { changes <- c }, func(err error) { failures <- err })
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = w.Close() })
	return w, changes, failures
}

// await receives one test event or fails after the bounded delivery timeout.
func await[T any](t *testing.T, ch <-chan T) T {
	t.Helper()
	select {
	case value := <-ch:
		return value
	case <-time.After(7 * time.Second):
		t.Fatal("watch callback timeout")
		var zero T
		return zero
	}
}

// awaitName reads shared Viper state under its lock until the expected source version arrives.
func awaitName(t *testing.T, p *Provider, changes <-chan provider.Change, name string) {
	t.Helper()
	deadline := time.After(7 * time.Second)
	for {
		select {
		case change := <-changes:
			p.ConfigMutex().RLock()
			matched := change.Config.GetString("name") == name
			p.ConfigMutex().RUnlock()
			if matched {
				return
			}
		case <-deadline:
			t.Fatalf("watch did not converge to %s", name)
		}
	}
}

// fakeProvider injects an isolated SDK fake and registers Provider resource cleanup.
func fakeProvider(t *testing.T, f *fakeSDK) *Provider {
	t.Helper()
	p := New(WithDataID("test.yaml"), WithServers([]constant.ServerConfig{{IpAddr: "localhost", Port: 8848}}), WithClientConfig(constant.ClientConfig{CacheDir: t.TempDir()}))
	p.factory = func(vo.NacosClientParam) (sdkClient, error) { return f, nil }
	t.Cleanup(func() { _ = p.Close() })
	return p
}

// TestWatcherReconciliationCoalescingAndCleanup checks subscription-gap recovery,
// coalesced wakeups, current-source reads, and subscription/cache cleanup.
func TestWatcherReconciliationCoalescingAndCleanup(t *testing.T) {
	f := &fakeSDK{content: "name: before"}
	p := fakeProvider(t, f)
	if _, err := p.Load(context.Background()); err != nil {
		t.Fatal(err)
	}
	dir := p.directory
	f.setContent("name: between-load-and-watch")
	w, changes, failures := watchEvents(t, p, context.Background())
	awaitName(t, p, changes, "between-load-and-watch")
	same, err := p.Watch(context.Background(), func(provider.Change) { t.Error("callbacks replaced") }, nil)
	if err != nil || same != w {
		t.Fatal("duplicate subscription", err)
	}
	f.setContent("name: latest")
	for i := 0; i < 1000; i++ {
		f.notify()
	}
	awaitName(t, p, changes, "latest")
	f.setContent("[]")
	f.notify()
	if err := await(t, failures); !errors.Is(err, ErrDocument) {
		t.Fatal(err)
	}
	f.setContent("name: recovered")
	f.notify()
	awaitName(t, p, changes, "recovered")
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	await(t, w.(*watcher).finished)
	_ = w.Close()
	_ = p.Close()
	_ = p.Close()
	f.mu.Lock()
	canceled, closed := f.canceled, f.closed
	f.mu.Unlock()
	if canceled != 1 || closed != 1 {
		t.Fatalf("cancel=%d close=%d", canceled, closed)
	}
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Fatal("cache directory not removed")
	}
}

// TestStrictCacheAndSafeErrors checks private cache ownership, strict startup,
// and errors that omit SDK configuration content.
func TestStrictCacheAndSafeErrors(t *testing.T) {
	f := &fakeSDK{readErr: errors.New("secret-content secret-password")}
	p := fakeProvider(t, f)
	parent := p.opts.client.CacheDir
	if err := os.WriteFile(filepath.Join(parent, "snapshot"), []byte("old"), 0600); err != nil {
		t.Fatal(err)
	}
	p.factory = func(param vo.NacosClientParam) (sdkClient, error) {
		c := param.ClientConfig
		if !c.DisableUseSnapShot || !c.NotLoadCacheAtStart || c.CacheDir == parent {
			t.Fatal("strict cache flags not applied")
		}
		info, err := os.Stat(c.CacheDir)
		if err != nil || info.Mode().Perm() != 0700 {
			t.Fatal("cache permissions")
		}
		if entries, err := os.ReadDir(c.CacheDir); err != nil || len(entries) != 1 || entries[0].Name() != "logs" {
			t.Fatal("old cache reused")
		}
		return f, nil
	}
	if _, err := p.Load(context.Background()); !errors.Is(err, ErrRead) || strings.Contains(err.Error(), "secret") {
		t.Fatal(err)
	}
}

// TestWatchFailureRetryAndContext checks registration failure, retry, and context cancellation.
func TestWatchFailureRetryAndContext(t *testing.T) {
	f := &fakeSDK{content: "name: x", listenErr: errors.New("secret")}
	p := fakeProvider(t, f)
	if _, err := p.Load(context.Background()); err != nil {
		t.Fatal(err)
	}
	if _, err := p.Watch(context.Background(), nil, nil); !errors.Is(err, ErrSubscribe) || strings.Contains(err.Error(), "secret") {
		t.Fatal(err)
	}
	f.listenErr = nil
	ctx, cancel := context.WithCancel(context.Background())
	w, changes, _ := watchEvents(t, p, ctx)
	f.setContent("name: watched")
	f.notify()
	awaitName(t, p, changes, "watched")
	cancel()
	await(t, w.(*watcher).finished)
	if _, err := p.Watch(ctx, nil, nil); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	f.setContent("name: restarted")
	_, changes, _ = watchEvents(t, p, context.Background())
	awaitName(t, p, changes, "restarted")
}

// TestWatcherReconcilesWithoutNotification verifies consistency reads recover a missed SDK notification.
func TestWatcherReconcilesWithoutNotification(t *testing.T) {
	f := &fakeSDK{content: "name: initial"}
	p := fakeProvider(t, f)
	_, changes, _ := watchEvents(t, p, context.Background())
	awaitName(t, p, changes, "initial")
	f.setContent("name: missed-notification")
	awaitName(t, p, changes, "missed-notification")
}

// TestWatchCallbackCanCloseProvider checks callback reentry can close without deadlocking.
func TestWatchCallbackCanCloseProvider(t *testing.T) {
	f := &fakeSDK{content: "name: initial"}
	p := fakeProvider(t, f)
	closed := make(chan error, 1)
	w, err := p.Watch(context.Background(), func(provider.Change) { closed <- p.Close() }, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := await(t, closed); err != nil {
		t.Fatal(err)
	}
	await(t, w.(*watcher).finished)
}

// TestWatchReconciliationReadFailureCanRetry checks a failed post-subscription read
// is reported, cleaned up, and retryable.
func TestWatchReconciliationReadFailureCanRetry(t *testing.T) {
	f := &fakeSDK{content: "name: initial"}
	p := fakeProvider(t, f)
	if _, err := p.Load(context.Background()); err != nil {
		t.Fatal(err)
	}
	f.readErr = errors.New("secret-read-failure")
	if _, err := p.Watch(context.Background(), nil, nil); !errors.Is(err, ErrRead) || strings.Contains(err.Error(), "secret") {
		t.Fatal(err)
	}
	if p.watcher != nil || f.canceled != 1 {
		t.Fatal("failed registration retained subscription")
	}
	f.readErr = nil
	f.setContent("name: after-retry")
	_, changes, _ := watchEvents(t, p, context.Background())
	awaitName(t, p, changes, "after-retry")
}

// Load and watch must update the Viper already handed to Confy.
func TestLoadAndWatchShareViper(t *testing.T) {
	f := &fakeSDK{content: "name: initial\nremoved: old"}
	p := fakeProvider(t, f)
	created := 0
	factory := p.factory
	p.factory = func(param vo.NacosClientParam) (sdkClient, error) {
		created++
		return factory(param)
	}
	initial, err := p.Load(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	f.setContent("name: loaded")
	loaded, err := p.Load(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if loaded != initial || initial.GetString("name") != "loaded" || initial.IsSet("removed") {
		t.Fatal("Load did not replace source data in the shared Viper")
	}
	type result struct {
		same    bool
		name    string
		removed bool
	}
	changes := make(chan result, 8)
	f.setContent("name: reconciled")
	w, err := p.Watch(context.Background(), func(change provider.Change) {
		changes <- result{change.Config == initial, change.Config.GetString("name"), change.Config.IsSet("removed")}
	}, func(err error) { t.Error(err) })
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()
	await(t, changes)
	f.setContent("name: watched")
	f.notify()
	got := await(t, changes)
	if !got.same || got.name != "watched" || got.removed {
		t.Fatalf("watch did not update the shared Viper before notification: %+v", got)
	}

	p.mu.Lock()
	clients := created
	p.mu.Unlock()
	if clients != 1 {
		t.Fatalf("created %d clients across loads and watch updates", clients)
	}
}

// TestConfyReadsLatestSourceWhenRebindingFails distinguishes the latest parseable Viper
// source from the last successfully bound target and verifies subsequent recovery.
func TestConfyReadsLatestSourceWhenRebindingFails(t *testing.T) {
	f := &fakeSDK{content: "name: initial\nport: 1\nremoved: old\nsecret: ''"}
	p := fakeProvider(t, f)
	t.Setenv("CONFYSHARED_SECRET", "from-env")
	type targetConfig struct {
		Name    string
		Port    int
		Removed string
		Secret  string
	}
	var target targetConfig
	updated := make(chan targetConfig, 8)
	failed := make(chan any, 8)
	var cfg *confy.Config
	cfg, err := confy.New(confy.WithProvider(p), confy.WithWatch(true), confy.WithEnvPrefix("CONFYSHARED"),
		confy.WithOnChange(func(confy.Event) { updated <- target }),
		confy.WithOnError(func(error) { failed <- cfg.Get("port") }))
	if err != nil {
		t.Fatal(err)
	}
	defer cfg.Close()
	cfg.Set("name", "explicit")
	if err := cfg.Bind(&target); err != nil {
		t.Fatal(err)
	}
	f.setContent("name: invalid\nport: not-an-int\nsecret: ''")
	f.notify()
	if got := await(t, failed); got != "not-an-int" {
		t.Fatalf("Get has stale source: %v", got)
	}
	if target.Port != 1 || target.Removed != "old" || cfg.Get("removed") != nil {
		t.Fatal("failed binding must keep target while Viper reflects the latest source")
	}
	f.setContent("name: recovered\nport: 3\nsecret: ''")
	f.notify()
	got := await(t, updated)
	if got.Name != "explicit" || got.Port != 3 || got.Removed != "" || got.Secret != "from-env" {
		t.Fatalf("source replacement lost binding, ENV or explicit override: %+v", got)
	}
}

// TestWriteConfigReplacesSourceAndPreservesOverrides checks stable Viper identity,
// removal of deleted source fields, and persistence of explicit Set values.
func TestWriteConfigReplacesSourceAndPreservesOverrides(t *testing.T) {
	for format, documents := range map[confy.FileType][2]string{
		confy.YAML: {"name: first\nremoved: old\nport: 1", "name: second\nport: 2"},
		confy.JSON: {`{"name":"first","removed":"old","port":1}`, `{"name":"second","port":2}`},
		confy.TOML: {"name='first'\nremoved='old'\nport=1", "name='second'\nport=2"},
	} {
		t.Run(string(format), func(t *testing.T) {
			p := New(WithFormat(format))
			ctx := context.Background()
			if err := p.writeConfig(ctx, documents[0]); err != nil {
				t.Fatal(err)
			}
			shared := p.v
			shared.SetEnvPrefix("CONFYWRITE")
			shared.AutomaticEnv()
			t.Setenv("CONFYWRITE_NAME", "env")
			shared.Set("port", 99)
			if err := p.writeConfig(ctx, documents[1]); err != nil {
				t.Fatal(err)
			}
			if p.v != shared || shared.IsSet("removed") || shared.GetString("name") != "env" || shared.GetInt("port") != 99 {
				t.Fatal("source replacement lost shared identity, deletion or overrides")
			}
			for _, invalid := range []string{"", "  ", "[]", "null", "{}", "# comment-only", "name: [broken"} {
				if err := p.writeConfig(ctx, invalid); !errors.Is(err, ErrDocument) {
					t.Fatalf("accepted %q: %v", invalid, err)
				}
				// Remove ENV temporarily to check the actual source survived the failure.
				t.Setenv("CONFYWRITE_NAME", "")
				if shared.GetString("name") != "second" {
					t.Fatalf("invalid document replaced source: %q", invalid)
				}
				t.Setenv("CONFYWRITE_NAME", "env")
			}
			canceled, cancel := context.WithCancel(ctx)
			cancel()
			if err := p.writeConfig(canceled, documents[0]); !errors.Is(err, context.Canceled) {
				t.Fatal(err)
			}
			if shared.IsSet("removed") {
				t.Fatal("canceled write replaced source")
			}
		})
	}
}

// TestWatcherOwnsDeduplicationAndIncludesOverrides checks suppression of unchanged
// reads while including effective Set changes in Provider notification decisions.
func TestWatcherOwnsDeduplicationAndIncludesOverrides(t *testing.T) {
	f := &fakeSDK{content: "name: initial", observed: make(chan string, 32)}
	p := fakeProvider(t, f)
	if _, err := p.Load(context.Background()); err != nil {
		t.Fatal(err)
	}
	await(t, f.observed)
	type config struct {
		Name string
		Port int
	}
	changes := make(chan config, 8)
	failures := make(chan error, 8)
	w, err := p.Watch(context.Background(), func(provider.Change) {
		p.ConfigMutex().RLock()
		value := config{p.v.GetString("name"), p.v.GetInt("port")}
		p.ConfigMutex().RUnlock()
		changes <- value
	}, func(err error) { failures <- err })
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()
	await(t, f.observed) // post-subscription read
	await(t, f.observed) // initial refresh must not notify unchanged settings
	f.setContent("name: next")
	f.notify()
	if got := await(t, changes); got != (config{Name: "next"}) {
		t.Fatalf("unchanged initial config notified: %+v", got)
	}
	await(t, f.observed)
	p.ConfigMutex().Lock()
	p.v.Set("name", "override")
	p.ConfigMutex().Unlock()
	f.notify()
	if got := await(t, changes); got != (config{Name: "override"}) {
		t.Fatal(got)
	}
	await(t, f.observed)
	f.notify()
	await(t, f.observed) // duplicate refresh; a distinct version is the barrier below
	f.setContent("name: next\nport: 2")
	f.notify()
	if got := await(t, changes); got != (config{Name: "override", Port: 2}) {
		t.Fatalf("duplicate notified: %+v", got)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	await(t, w.(*watcher).finished)
	select {
	case err := <-failures:
		t.Fatal(err)
	default:
	}
}

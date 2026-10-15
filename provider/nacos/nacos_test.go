package nacos

import (
	"context"
	"errors"
	"github.com/nacos-group/nacos-sdk-go/v2/common/constant"
	"github.com/nacos-group/nacos-sdk-go/v2/vo"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/JsonLee12138/confy"
)

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
			if _, err := parse(content, format); err == nil {
				t.Fatalf("accepted %s %q", format, content)
			}
		}
	}
	for format, content := range map[confy.FileType]string{confy.YAML: "name: test", confy.JSON: `{"name":"test"}`, confy.TOML: `name = "test"`} {
		v, err := parse(content, format)
		if err != nil || v.GetString("name") != "test" {
			t.Fatalf("%s: %v", format, err)
		}
	}
}

type fakeSDK struct {
	content                 string
	readErr, listenErr      error
	callback                vo.Listener
	reads, canceled, closed int
}

func (f *fakeSDK) GetConfig(vo.ConfigParam) (string, error) { f.reads++; return f.content, f.readErr }
func (f *fakeSDK) ListenConfig(p vo.ConfigParam) error      { f.callback = p.OnChange; return f.listenErr }
func (f *fakeSDK) CancelListenConfig(vo.ConfigParam) error  { f.canceled++; return nil }
func (f *fakeSDK) CloseClient()                             { f.closed++ }
func fakeProvider(t *testing.T, f *fakeSDK) *Provider {
	t.Helper()
	p := New(WithDataID("test.yaml"), WithServers([]constant.ServerConfig{{IpAddr: "localhost", Port: 8848}}), WithClientConfig(constant.ClientConfig{CacheDir: t.TempDir()}))
	p.factory = func(vo.NacosClientParam) (sdkClient, error) { return f, nil }
	t.Cleanup(func() { _ = p.Close() })
	return p
}
func TestWatcherReconciliationCoalescingAndCleanup(t *testing.T) {
	f := &fakeSDK{content: "name: before"}
	p := fakeProvider(t, f)
	if _, err := p.Load(context.Background()); err != nil {
		t.Fatal(err)
	}
	dir := p.directory
	f.content = "name: between-load-and-watch"
	w, err := p.Watch(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	change, err := w.Next(context.Background())
	if err != nil || change.Config.GetString("name") != "between-load-and-watch" {
		t.Fatal(err)
	}
	f.content = "name: latest"
	for i := 0; i < 1000; i++ {
		f.callback("", "", "", "out-of-order-old-content")
	}
	change, err = w.Next(context.Background())
	if err != nil || change.Config.GetString("name") != "latest" {
		t.Fatal(err)
	}
	f.content = "[]"
	f.callback("", "", "", "")
	if _, err = w.Next(context.Background()); !errors.Is(err, ErrDocument) {
		t.Fatal(err)
	}
	f.content = "name: recovered"
	f.callback("", "", "", "")
	if _, err = w.Next(context.Background()); err != nil {
		t.Fatal(err)
	}
	result := make(chan error, 1)
	go func() { _, err := w.Next(context.Background()); result <- err }()
	if err = w.Close(); err != nil {
		t.Fatal(err)
	}
	select {
	case err = <-result:
		if !errors.Is(err, context.Canceled) {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("Next remained blocked")
	}
	_ = w.Close()
	_ = p.Close()
	_ = p.Close()
	if f.canceled != 1 || f.closed != 1 {
		t.Fatalf("cancel=%d close=%d", f.canceled, f.closed)
	}
	if _, err = os.Stat(dir); !os.IsNotExist(err) {
		t.Fatal("cache directory not removed")
	}
}
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
func TestWatchFailureRetryAndContext(t *testing.T) {
	f := &fakeSDK{content: "name: x", listenErr: errors.New("secret")}
	p := fakeProvider(t, f)
	if _, err := p.Load(context.Background()); err != nil {
		t.Fatal(err)
	}
	if _, err := p.Watch(context.Background()); !errors.Is(err, ErrSubscribe) || strings.Contains(err.Error(), "secret") {
		t.Fatal(err)
	}
	f.listenErr = nil
	w, err := p.Watch(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if _, err = w.Next(context.Background()); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err = w.Next(ctx); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	_ = w.Close()
	if _, err = p.Watch(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func TestWatcherReconcilesWithoutNotification(t *testing.T) {
	f := &fakeSDK{content: "name: initial"}
	p := fakeProvider(t, f)
	if _, err := p.Load(context.Background()); err != nil {
		t.Fatal(err)
	}
	w, err := p.Watch(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()
	if _, err = w.Next(context.Background()); err != nil {
		t.Fatal(err)
	}
	f.content = "name: missed-notification"
	ctx, cancel := context.WithTimeout(context.Background(), 7*time.Second)
	defer cancel()
	change, err := w.Next(ctx)
	if err != nil || change.Config.GetString("name") != "missed-notification" {
		t.Fatal("consistency reconciliation failed", err)
	}
}
func TestWatchContextUnblocksNext(t *testing.T) {
	f := &fakeSDK{content: "name: initial"}
	p := fakeProvider(t, f)
	ctx, cancel := context.WithCancel(context.Background())
	w, err := p.Watch(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()
	if _, err = w.Next(context.Background()); err != nil {
		t.Fatal(err)
	}
	cancel()
	if _, err = w.Next(context.Background()); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
}

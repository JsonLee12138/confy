//go:build nacos_integration

package nacos

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/JsonLee12138/confy"
	"github.com/nacos-group/nacos-sdk-go/v2/common/constant"
)

type integrationServer struct {
	url, namespace, group, token string
	client                       *http.Client
	server                       constant.ServerConfig
}

func integrationSettings(t *testing.T) *integrationServer {
	t.Helper()
	if os.Getenv("NACOS_INTEGRATION") != "1" {
		t.Fatal("explicit integration run requires NACOS_INTEGRATION=1")
	}
	for _, key := range []string{"NACOS_SERVER_ADDR", "NACOS_NAMESPACE_ID", "NACOS_TEST_GROUP"} {
		if _, ok := os.LookupEnv(key); !ok {
			t.Fatalf("missing %s", key)
		}
	}
	addr := os.Getenv("NACOS_SERVER_ADDR")
	host, portText, err := net.SplitHostPort(addr)
	if err != nil {
		t.Fatal("NACOS_SERVER_ADDR must be host:port")
	}
	port, err := strconv.ParseUint(portText, 10, 64)
	if err != nil {
		t.Fatal("invalid port")
	}
	if os.Getenv("NACOS_TEST_GROUP") == "" {
		t.Fatal("test group cannot be empty")
	}
	s := &integrationServer{url: "http://" + addr, namespace: os.Getenv("NACOS_NAMESPACE_ID"), group: os.Getenv("NACOS_TEST_GROUP"), client: &http.Client{Timeout: 5 * time.Second}, server: constant.ServerConfig{IpAddr: host, Port: port}}
	if user := os.Getenv("NACOS_USERNAME"); user != "" {
		resp, err := s.client.PostForm(s.url+"/nacos/v1/auth/login", url.Values{"username": {user}, "password": {os.Getenv("NACOS_PASSWORD")}})
		if err != nil {
			t.Fatal("fixture login failed")
		}
		defer resp.Body.Close()
		var result struct {
			AccessToken string `json:"accessToken"`
		}
		if json.NewDecoder(resp.Body).Decode(&result) != nil || result.AccessToken == "" {
			t.Fatal("fixture login rejected")
		}
		s.token = result.AccessToken
	}
	return s
}
func (s *integrationServer) request(method, id, content string) (int, string, error) {
	values := url.Values{"dataId": {id}, "group": {s.group}, "tenant": {s.namespace}, "content": {content}, "type": {"yaml"}}
	if s.token != "" {
		values.Set("accessToken", s.token)
	}
	endpoint := s.url + "/nacos/v1/cs/configs"
	var body io.Reader
	if method == http.MethodPost {
		body = strings.NewReader(values.Encode())
	} else {
		endpoint += "?" + values.Encode()
	}
	req, err := http.NewRequest(method, endpoint, body)
	if err != nil {
		return 0, "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := s.client.Do(req)
	if err != nil {
		return 0, "", err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	return resp.StatusCode, string(data), err
}
func (s *integrationServer) publish(t *testing.T, id, content string) {
	t.Helper()
	status, body, err := s.request(http.MethodPost, id, content)
	if err != nil || status != 200 || body != "true" {
		t.Fatal("isolated fixture publish failed")
	}
}
func encryptedFixture(t *testing.T, plain string, key []byte) string {
	t.Helper()
	block, err := aes.NewCipher(key)
	if err != nil {
		t.Fatal(err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		t.Fatal(err)
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err = rand.Read(nonce); err != nil {
		t.Fatal(err)
	}
	return "enc:AES_GCM:" + base64.StdEncoding.EncodeToString(gcm.Seal(nonce, nonce, []byte(plain), nil))
}

type integrationConfig struct {
	Server struct {
		Port int `mapstructure:"port"`
	} `mapstructure:"server"`
	Name      string
	Removed   string
	Port      int
	Password  string
	Encrypted string
	Timeout   int `default:"30"`
}

func (c *integrationConfig) Validate() error {
	if c.Port < 0 || c.Name == "invalid-validator" {
		return fmt.Errorf("rejected")
	}
	return nil
}
func awaitConfig(t *testing.T, ch <-chan integrationConfig, name string) integrationConfig {
	t.Helper()
	timer := time.NewTimer(60 * time.Second)
	defer timer.Stop()
	for {
		select {
		case cfg := <-ch:
			if cfg.Name == name {
				return cfg
			}
		case <-timer.C:
			t.Fatalf("remote did not converge to %s", name)
		}
	}
}
func TestIntegrationRNacosLifecycle(t *testing.T) {
	s := integrationSettings(t)
	id := fmt.Sprintf("confy-integration-%d.yaml", time.Now().UnixNano())
	t.Cleanup(func() {
		status, _, err := s.request(http.MethodDelete, id, "")
		if err != nil || status != 200 {
			t.Error("fixture cleanup failed")
		}
	})
	key := []byte("01234567890123456789012345678901")
	plain := "confy-test-secret-7b8d0c"
	encoded := encryptedFixture(t, plain, key)
	content := func(name string, port int) string {
		return fmt.Sprintf("server:\n  port: 8080\nname: %s\nport: %d\npassword: ''\nencrypted: %s\n", name, port, encoded)
	}
	s.publish(t, id, content("initial", 1)+"removed: initial\n")
	t.Setenv("CONFYIT_PASSWORD", plain)
	t.Setenv("CONFYIT_SERVER_PORT", "9090")
	httpProxy := newDisconnectProxy(t, net.JoinHostPort(s.server.IpAddr, strconv.FormatUint(s.server.Port, 10)))
	grpcProxy := newDisconnectProxy(t, net.JoinHostPort(s.server.IpAddr, strconv.FormatUint(s.server.Port+1000, 10)))
	server := s.server
	server.IpAddr = "127.0.0.1"
	server.Port = httpProxy.port()
	server.GrpcPort = grpcProxy.port()
	cacheParent := t.TempDir()
	p := New(WithDataID(id), WithGroup(s.group), WithServers([]constant.ServerConfig{server}), WithClientConfig(constant.ClientConfig{NamespaceId: s.namespace, Username: os.Getenv("NACOS_USERNAME"), Password: os.Getenv("NACOS_PASSWORD"), TimeoutMs: 2000, CacheDir: cacheParent}))
	updates := make(chan integrationConfig, 64)
	rejected := make(chan error, 64)
	var bound integrationConfig
	cfg, err := confy.New(confy.WithProvider(p), confy.WithEnvPrefix("CONFYIT"), confy.WithEncryption("aes-gcm", key), confy.WithWatch(true), confy.WithOnChange(func(confy.Event) { t.Logf("accepted name=%s", bound.Name); updates <- bound }), confy.WithOnError(func(err error) { t.Logf("rejected: %v", err); rejected <- err }))
	if err != nil {
		t.Fatal(err)
	}
	defer cfg.Close()
	if cfg.Get("server.port") != "9090" {
		t.Fatal("initial Nacos read did not apply nested ENV override")
	}
	// Change after Load, before the first subscription.
	s.publish(t, id, content("between", 2))
	if err = cfg.BindWithDefaults(&bound); err != nil {
		t.Fatal(err)
	}
	got := awaitConfig(t, updates, "between")
	if got.Password != plain || got.Encrypted != plain || got.Removed != "" || got.Timeout != 30 || got.Server.Port != 9090 {
		t.Fatal("binding/ENV/decryption/deletion mismatch")
	}
	cfg.Set("port", 99)
	for i := 0; i < 4; i++ {
		s.publish(t, id, content(fmt.Sprintf("burst-%d", i), 3+i))
	}
	got = awaitConfig(t, updates, "burst-3")
	if got.Port != 99 || got.Server.Port != 9090 {
		t.Fatal("Set override lost")
	}
	currentName := "burst-3"
	for i, invalid := range []string{"name: [broken", "name: invalid\nencrypted: enc:AES_GCM:broken", "name: invalid-type\ntimeout: not-an-int", "name: invalid-validator", "  "} {
		// A successful intervening version is a serial-consumer barrier. Drain
		// prior rejections so each error assertion belongs to this publication.
	drain:
		for {
			select {
			case <-rejected:
			default:
				break drain
			}
		}
		s.publish(t, id, invalid)
		select {
		case <-rejected:
		case <-time.After(15 * time.Second):
			t.Fatal("rejected update not reported")
		}
		if cfg.Get("name") != currentName {
			t.Fatal("invalid update committed")
		}
		currentName = fmt.Sprintf("valid-%d", i)
		s.publish(t, id, content(currentName, 20+i))
		awaitConfig(t, updates, currentName)
	}
	if status, _, err := s.request(http.MethodDelete, id, ""); err != nil || status != 200 {
		t.Fatal("fixture delete failed")
	}
	select {
	case <-rejected:
	case <-time.After(15 * time.Second):
		t.Fatal("deleted Data ID not rejected")
	}
	if cfg.Get("name") != currentName {
		t.Fatal("delete cleared config")
	}
	s.publish(t, id, content("recovered", 8))
	awaitConfig(t, updates, "recovered")
	// Break only test-owned connections, leaving the shared rnacos untouched.
	httpProxy.disconnect(true)
	grpcProxy.disconnect(true)
	s.publish(t, id, content("offline", 9))
	if cfg.Get("name") != "recovered" {
		t.Fatal("disconnect changed effective config")
	}
	httpProxy.disconnect(false)
	grpcProxy.disconnect(false)
	if got := awaitConfig(t, updates, "offline"); got.Server.Port != 9090 {
		t.Fatal("reconnected update lost nested ENV override")
	}
	status, raw, err := s.request(http.MethodGet, id, "")
	if err != nil || status != 200 || !strings.Contains(raw, "name: offline") {
		t.Fatal("remote readback failed")
	}
	// ENV and Confy-decrypted values must never be written into SDK raw cache.
	if err = filepath.WalkDir(cacheParent, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			data, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			if strings.Contains(string(data), plain) {
				t.Error("plaintext secret appeared in SDK files")
			}
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if os.Getenv("NACOS_EXPECT_AUTH") == "1" {
		unauth := *s
		unauth.token = ""
		status, _, err = unauth.request(http.MethodGet, id, "")
		if err != nil || (status != 401 && status != 403) {
			t.Fatal("server did not deny unauthenticated read")
		}
		bad := New(WithDataID(id), WithGroup(s.group), WithServers([]constant.ServerConfig{s.server}), WithClientConfig(constant.ClientConfig{NamespaceId: s.namespace, Username: "confy-invalid-user", Password: "confy-invalid-password", TimeoutMs: 1000, CacheDir: t.TempDir()}))
		defer bad.Close()
		if _, err = bad.Load(context.Background()); err == nil {
			t.Fatal("invalid SDK credentials accepted")
		}
	} else {
		t.Log("AUTH_NOT_VERIFIED: target OpenAPI auth is not required; TLS_NOT_VERIFIED: target is plaintext")
	}
	if err = cfg.Close(); err != nil {
		t.Fatal(err)
	}
	if entries, err := os.ReadDir(cacheParent); err != nil || len(entries) != 0 {
		t.Fatal("owned cache not cleaned")
	}
	t.Logf("readback: name=offline port(source)=9 port(effective)=99; HTTP=%d gRPC=%d; Data ID cleaned on exit", s.server.Port, s.server.Port+1000)
}

// disconnectProxy is an isolated fault fixture, never a server control action.
type disconnectProxy struct {
	mu       sync.Mutex
	listener net.Listener
	target   string
	blocked  bool
	conns    map[net.Conn]struct{}
}

func newDisconnectProxy(t *testing.T, target string) *disconnectProxy {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	p := &disconnectProxy{listener: l, target: target, conns: make(map[net.Conn]struct{})}
	t.Cleanup(func() { _ = l.Close(); p.disconnect(true) })
	go func() {
		for {
			incoming, err := l.Accept()
			if err != nil {
				return
			}
			go p.forward(incoming)
		}
	}()
	return p
}
func (p *disconnectProxy) port() uint64 { return uint64(p.listener.Addr().(*net.TCPAddr).Port) }
func (p *disconnectProxy) forward(in net.Conn) {
	out, err := net.DialTimeout("tcp", p.target, 2*time.Second)
	if err != nil {
		_ = in.Close()
		return
	}
	p.mu.Lock()
	if p.blocked {
		p.mu.Unlock()
		_ = in.Close()
		_ = out.Close()
		return
	}
	p.conns[in] = struct{}{}
	p.conns[out] = struct{}{}
	p.mu.Unlock()
	defer func() {
		_ = in.Close()
		_ = out.Close()
		p.mu.Lock()
		delete(p.conns, in)
		delete(p.conns, out)
		p.mu.Unlock()
	}()
	done := make(chan struct{})
	go func() { _, _ = io.Copy(out, in); _ = out.Close(); close(done) }()
	_, _ = io.Copy(in, out)
	_ = in.Close()
	<-done
}
func (p *disconnectProxy) disconnect(block bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.blocked = block
	if block {
		for conn := range p.conns {
			_ = conn.Close()
		}
	}
}

func TestIntegrationStrictStartupWithOldCache(t *testing.T) {
	s := integrationSettings(t)
	parent := t.TempDir()
	if err := os.MkdirAll(filepath.Join(parent, "config"), 0700); err != nil {
		t.Fatal(err)
	}
	for _, suffix := range []string{"", "_failover"} {
		if err := os.WriteFile(filepath.Join(parent, "config", "confy-strict@@"+s.group+"@@"+s.namespace+suffix), []byte("name: old-cache"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	proxy := newDisconnectProxy(t, net.JoinHostPort(s.server.IpAddr, strconv.FormatUint(s.server.Port, 10)))
	proxy.disconnect(true)
	grpc := newDisconnectProxy(t, net.JoinHostPort(s.server.IpAddr, strconv.FormatUint(s.server.Port+1000, 10)))
	grpc.disconnect(true)
	p := New(WithDataID("confy-strict"), WithGroup(s.group), WithServers([]constant.ServerConfig{{IpAddr: "127.0.0.1", Port: proxy.port(), GrpcPort: grpc.port()}}), WithClientConfig(constant.ClientConfig{CacheDir: parent, NamespaceId: s.namespace, TimeoutMs: 500}))
	if cfg, err := confy.New(confy.WithProvider(p)); err == nil {
		_ = cfg.Close()
		t.Fatal("unavailable remote started from old cache")
	}
	entries, err := os.ReadDir(parent)
	if err != nil || len(entries) != 1 || entries[0].Name() != "config" {
		t.Fatal("failed New leaked owned SDK files")
	}
	t.Log("strict startup rejected unavailable remote despite pre-existing snapshot and failover; owned files removed")
}

# Nacos adapter operational contract

Both adapters implement the canonical `core.Provider` and `core.WatchProvider` contracts. NacosProvider returns `core.ProviderNacos` from Type; the shared coordinator uses explicit provider-type dispatch. Production adapter code depends on core, not the root confy package. The root confy package does not import the SDK. This adapter uses official `nacos-sdk-go/v2` pinned to `v2.3.6-0.20260917093256-70993e7b0590` (commit `70993e7b0590aee5299849315d01a92fd346f8ab`). This is an immutable upstream commit, not a floating master dependency. The tagged v2.3.5 release was rejected after a real reconnect race test; the pinned revision includes upstream atomic connection/state fixes. The module dependency graph includes the SDK's Alibaba/KMS dependencies even though this adapter rejects OpenKMS and does not provide KMS functionality.

`New` collects functional options. `Load` creates the client and strictly reads the remote object. YAML is the default; JSON and TOML require WithFormat. Data ID extensions do not choose a parser. Empty/whitespace/scalar/sequence/empty-object documents are rejected. Group defaults to DEFAULT_GROUP; the public namespace is the empty SDK namespace ID. Namespace/Group/Data ID cannot contain filesystem separators or control characters because the SDK also uses them as cache file names. Remote `base` is an ordinary field, never a local inheritance path.

SDK callbacks only coalesce wakeups in a bounded channel. Next re-reads the current remote value, so reordered SDK callback goroutines cannot commit stale callback content. Registration includes a post-subscription remote read and initial reconciliation. Failed reads retry with a one-second delay; an idle watcher performs an SDK consistency read every five seconds to recover missed/coalesced notifications. SDK owns subscription, authentication and transport reconnection; no custom long-poll protocol is implemented. Updates may coalesce and do not guarantee every intermediate version. Config owns the SDK client; there is no borrowed-client API.

Strict startup uses a fresh 0700 temporary directory for every Provider, with snapshot fallback disabled. `ClientConfig.CacheDir`, if set, is the **parent** of that private directory; existing snapshots and failover are never reused. Close and failed confy.New remove owned files. The SDK still persists **raw Nacos source content** during the Config lifetime: literal secrets stored in Nacos may therefore appear in this private cache. Use empty keys plus ENV/Secrets or Confy field ciphertext. This is not a zero-disk or memory-only SDK. A process crash can leave a private temporary directory requiring operational cleanup; it will never be used by a new Provider.

The SDK logger is process-global and logs raw configuration at info level and on some cache-write errors. The adapter installs a non-recording SDK logger on its first Load, also silencing other official SDK clients in the process. `ClientConfig.LogDir`, LogLevel and AppendToStdout do not override this safeguard; the adapter uses an owned private log path. Applications must not replace the SDK logger with a content-recording logger after initialization if they require this protection. Confy returns only safe phase/resource errors and uses WithOnError for update failures. SDK internal retry status is not fully observable through WithOnError. No SDK logger contains decoded ENV/Confy secrets in the integration fixture.

TLS is delegated to SDK ClientConfig.TLSCfg and ServerConfig.Scheme, with TrustAll rejected. HTTP and gRPC certificate verification and authorization require deployment-specific testing. The local r-nacos fixture has neither OpenAPI auth enforcement nor TLS, so those production checks remain unverified.

Run isolated integration fixtures (never existing business Data IDs):

```sh
export NACOS_INTEGRATION=1
export NACOS_SERVER_ADDR=127.0.0.1:8848
export NACOS_NAMESPACE_ID= # explicit empty value selects public
export NACOS_TEST_GROUP=CONFY_INTEGRATION
# Inject NACOS_USERNAME/NACOS_PASSWORD securely if required.
# Set NACOS_EXPECT_AUTH=1 only on an auth-enforcing test environment.
env -u GOROOT go test -tags=nacos_integration -count=1 -timeout=180s -v ./provider/nacos/...
env -u GOROOT go test -race -tags=nacos_integration -count=1 -timeout=180s -v ./provider/nacos/...
```

The current fixture supports plain HTTP/gRPC, not a TLS test environment. It creates unique confy-integration Data IDs, publishes and deletes only those, scans owned cache files for a test secret, and faults only its own proxy connections. An explicit tagged run missing required environment variables fails. Normal go test never accesses a server.

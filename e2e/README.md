# Cross-package E2E tests

These tests run as the external `e2e_test` package. They import `confy` and its Provider packages through public APIs, matching how an application consumes the library. Package-specific unit tests remain beside their implementations.

| File | Coverage |
| --- | --- |
| `local_test.go` | Real local files, mixed-format inheritance/loading/reload, binding, ENV/Set precedence, watching, failures and recovery |
| `env_test.go` | Process ENV, direct BindEnv, ENV Provider binding, defaults, errors and recovery |
| `time_test.go` | ENV direct/Provider time binding, pointers, invalid values, duration and weak-slice compatibility |
| `nacos_test.go` | Real Nacos lifecycle, reconnect, strict startup, cache cleanup, YAML/JSON/TOML updates |
| `files_test.go` | Fixed-size, single-write file fixtures without intermediate truncation |
| `settings_test.go` | Shared application configuration schema |

Run Local/ENV tests without external services:

```sh
go test -race -count=1 -v ./e2e
```

They also run as part of `go test ./...`. Real Nacos tests are opt-in and require a server:

```sh
export NACOS_INTEGRATION=1
export NACOS_SERVER_ADDR=127.0.0.1:8848
export NACOS_NAMESPACE_ID=
export NACOS_TEST_GROUP=CONFY_INTEGRATION
# Supply NACOS_USERNAME and NACOS_PASSWORD through your environment.
go test -race -tags=nacos_integration -count=1 -timeout=180s -v ./e2e -run '^TestIntegration'
```

The public namespace uses an explicitly empty namespace ID. Missing settings fail an explicit integration run. Test configurations use unique Data IDs in the selected test Group and are deleted on exit. Do not point the tests at business Data IDs. Connection interruption affects only test-owned TCP proxies.

See [Provider verification](../provider/TESTING.md) for tested behavior and environment limits. If your environment sets an incompatible GOROOT, prefix commands with `env -u GOROOT`.

Local watch fixtures use padded, same-size in-place writes. `os.WriteFile` truncates before writing and may expose a valid empty source between those operations on Linux. These tests assert behavior for a completed invalid document; they do not promise suppression of intermediate states written by arbitrary editors. The invalid-document preservation assertions remain enabled.

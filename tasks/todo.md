# Implementation checklist

- [x] Baseline and SDK/r-nacos preflight: existing `go test ./...`, SDK source/cache review, server version.
- [x] Provider loading/options: invalid result/conflict tests and unchanged local regression suite.
- [x] Watch coordination: retry, close, ENV/Set, deletion, rollback, callback reentry and local atomic replacement tests.
- [x] Nacos adapter: format/options, strict cache, bounded queue, cancellation and subscription reconciliation tests.
- [x] Real r-nacos: unique isolated fixtures, load/update/delete/recovery/readback; record auth/TLS limits.
- [x] Documentation and final checks: root build/test/race/vet/dependency isolation and nested example build/test.

## 未完成的发布条件

- [ ] 在要求鉴权的独立环境验证允许/拒绝（当前本机 OpenAPI 未要求鉴权）。
- [ ] 验证 HTTP/gRPC TLS 与证书链（当前服务是明文连接）。
- [ ] 处理依赖漏洞扫描报告，并确认所需 Go 基线升级；本次未作安全发布放行。

证据与代码行号见 `tasks/verification.md`。

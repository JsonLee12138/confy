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

## 本轮结构与示例调整

- [x] core 契约及根包兼容别名，Local/Nacos 明确类型与 switch 分派。
- [x] 清理 Config 冗余快照字段，归组监听与关闭状态，保留旧 API。
- [x] 恢复原 main，新建无 Viper/defaults 的 Nacos 读取示例。
- [x] 类型分派、ENV/热更新与全部回归、示例构建、真实 r-nacos 验证。

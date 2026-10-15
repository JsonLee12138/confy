# Local / Nacos implementation

Authorized by the user's 2026-10-15 implementation request; `docs/SPEC.md` is the acceptance contract. Preserve existing local discovery, inheritance, ENV and crypto behavior. Keep SDK imports in `provider/nacos`.

1. Establish local regression baseline; verify installed r-nacos and SDK strict-read/cache behavior.
2. Add Provider contracts/options and extract local loading (AC-01–04,18).
3. Add closable local watcher and serialized candidate commit/binding lifecycle (AC-05–13,15).
4. Add official SDK adapter, isolated ephemeral cache, bounded notifications and post-subscribe reconciliation (AC-14,16–19).
5. Test against localhost r-nacos using unique test Data IDs; record actual capabilities and any unverified auth/TLS/reconnect checks (AC-20).
6. Update usage docs/examples and run build, unit/race tests, vet and nested example checks.

Risks: SDK may cache plaintext or log raw content; verify SDK source and suppress its logs. Context cannot interrupt SDK calls; use SDK timeouts. External bound struct reads require application synchronization. Never mutate existing business Data IDs or restart the shared server for testing.

Changes remain reviewable on `codex/nacos-provider`; preserve pre-existing user changes. Tasks tracked in `tasks/todo.md`.

## 2026-10-15 用户要求调整

用户本轮要求优先于原 Spec 中的位置限制：契约移入 core，保留根包类型别名兼容已有调用；Provider 显式声明类型，公共层用 switch 选择对应策略，不以“非 Local 即 remote”推断。公共 ENV/解码能力继续共用。恢复旧 example/main.go，新增独立只读 Nacos 示例，示例不直接引用 Viper/defaults。清理 Config 只写不读状态、归组监听/关闭生命周期状态。

验证：旧 main 与 5333862 精确比较；核心/适配器契约编译断言；Provider 类型分派、ENV 首次读取及热更新；原测试、race、build/vet；example 独立构建；本机 r-nacos 集成。

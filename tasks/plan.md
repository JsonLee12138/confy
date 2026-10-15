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

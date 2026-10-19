# 公共辅助功能与兼容类型

Provider、WatchProvider、Watcher、Change、Event、FileType 的统一定义位于 [`provider/`](../provider/README.md)。`core/types.go` 保留文件类型和事件类型别名；Watcher 已采用回调式 Watch 和幂等 Close，自定义旧 Next 实现需要调整方法签名。

`core` 提供 ENV 覆盖辅助方法、模板继承及公共错误。结构体 ENV 绑定及 ENV Provider 位于 [`provider/env`](../provider/env/README.md)，也可通过根包 `confy.BindEnv` 直接绑定。所有通用错误在 `core/errors.go` 定义，根包通过别名暴露同一错误值。Local 和 Nacos 通过 provider 契约接入；Provider 负责刷新与 Config 共用的 Viper；根包 Bind 负责 defaults、解密与重新绑定；业务校验统一由显式 Validate 执行。监听 Provider 通过 ConfigMutex 与 Config 共享同步锁，不再有候选提交层。

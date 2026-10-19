# 三种 Provider 示例

在本目录运行，三个程序分别位于独立文件中：

| Provider | 文件 | 运行 |
| --- | --- | --- |
| Local | [local/main.go](local/main.go) | `go run ./local` |
| ENV | [env/main.go](env/main.go) | `MYAPP_SERVER_HOST=127.0.0.1 MYAPP_SERVER_PORT=9090 go run ./env` |
| Nacos | [nacos/main.go](nacos/main.go) | `NACOS_SERVER_ADDR=127.0.0.1:8848 NACOS_GROUP=CONFY_EXAMPLES NACOS_DATA_ID=confy-example.yaml go run ./nacos` |

Nacos 登录凭据通过 `NACOS_USERNAME` 和 `NACOS_PASSWORD` 环境变量注入；示例不会输出凭据。配置格式见 [Nacos 说明](nacos/README.md)。三个示例都用当前的 Bind；ENV 另演示直接 confy.BindEnv。Local/ENV 示例不依赖 Nacos。

Local 从现有 config/ 读取配置及继承，MYAPP_SERVER_PORT 可以覆盖端口。Nacos 示例只读取远程配置，不写回。原 main.go 是历史综合示例，保留原样。

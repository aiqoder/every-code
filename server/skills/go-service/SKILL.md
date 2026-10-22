---
name: Go 服务
description: 用 Go 写后端服务、命令行工具或 HTTP 接口时使用。约束工程结构、错误处理、并发与测试写法，避免写出只有壳子的代码。
---

# Go 服务

## 工程形态

新建服务时按下面结构落文件，不要把全部逻辑塞进 `main.go`。

```text
cmd/server/main.go        程序入口，只做装配与启动
internal/config/          配置结构与校验
internal/model/           数据结构
internal/repository/      数据访问
internal/logic/           业务逻辑
internal/handler/         HTTP 入口，只做参数解析与错误翻译
go.mod
```

- 模块名用仓库路径，不要用 `main` 或裸单词。
- 配置从环境变量或配置文件读取，密钥不进仓库。
- 可执行文件构建产物不进仓库。

## 必守规则

1. **错误必须处理。** 不允许 `_ = doSomething()` 吞掉可能失败的操作；实在要忽略，写清理由。
2. **包一层上下文。** 返回错误用 `fmt.Errorf("加载配置: %w", err)`，保留调用链。
3. **并发要能停。** 起 goroutine 必须能通过 `context` 或 `close(ch)` 退出，不允许裸 `for {}`。
4. **不猜依赖。** 引入第三方库前先确认 `go.mod` 里有没有；没有就说明理由再 `go get`。
5. **HTTP 处理函数不写业务。** 解析参数 → 调 logic → 写响应，三步结束。

## 常见写法

```go
// 带超时的外部调用
ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
defer cancel()

// 错误包装
if err := repo.Save(ctx, item); err != nil {
    return fmt.Errorf("保存订单 %d: %w", item.ID, err)
}
```

## 验收

- `go build ./...` 通过。
- `go vet ./...` 无告警。
- 有测试时 `go test ./...` 通过；新逻辑至少给关键分支写表驱动测试。
- 服务能起来，`curl` 打一下健康检查或主要接口有正常响应。

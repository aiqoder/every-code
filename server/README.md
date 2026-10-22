# chengzi-agent-server

基于 **Gin + GORM + Viper + 手写依赖注入** 的 Go API，配套 **Vue 3 管理后台**（`web/`）。

客户端已拆为独立仓库（与本仓互不共享源码依赖）：
- [`chengzi-agent-desktop`](../chengzi-agent-desktop)（Wails 桌面）
- [`chengzi-agent-mobile`](../chengzi-agent-mobile)（Capacitor 手机控制台）

> 无界面客户端 `chengzi-agent-client` 已暂时移除，执行面请以 `chengzi-agent-desktop` 为准。

自研桌面 / 客户端 / 手机应用请先读：**[自定义客户端开发指南](docs/client-dev-guide.md)**。

## 技术栈

| 组件 | 选型 |
|------|------|
| HTTP | Gin |
| ORM | GORM（MySQL / 纯 Go SQLite） |
| 配置 | Viper（`configs/config.yaml`） |
| DI | 手写构造函数注入（`internal/app`） |
| 管理端 | Vue 3 + Vite + Element Plus + Pinia（`web/`） |

## 目录结构

```
chengzi-agent-server/
├── cmd/server/           # 入口
├── configs/config.yaml   # 默认配置
├── internal/
│   ├── app/              # 手写 DI 装配与生命周期
│   ├── config/           # Viper 加载
│   ├── database/         # GORM 初始化
│   ├── handler/          # HTTP 层
│   ├── middleware/
│   ├── model/
│   ├── repository/       # 数据访问
│   └── service/          # 业务逻辑
├── pkg/response/         # 统一 JSON 信封
├── web/                  # Vue3 管理后台
├── Makefile
└── build.sh
```

## 快速开始

### 后端

```bash
# 按需修改 configs/config.yaml；默认可用 sqlite，生产建议 MySQL

# 开发热重启（修改 .go / configs/*.yaml 自动重编译启动）
make dev
# 首次需安装: go install github.com/air-verse/air@latest

# 或一次性启动（无热重启）
make run
```

默认监听 `8182`（见 `configs/config.yaml`）。

健康检查：

```bash
curl http://127.0.0.1:8182/api/health
```

创建用户示例：

```bash
curl -X POST http://127.0.0.1:8182/api/users \
  -H 'Content-Type: application/json' \
  -d '{"name":"demo","email":"demo@example.com"}'
```

### 管理后台

```bash
cd web
npm install
npm run dev
```

浏览器打开 Vite 提示地址（默认 `http://127.0.0.1:5173`）。开发环境将 `/api` 代理到后端 `8182`。

## 配置

唯一配置源：`configs/config.yaml`（可用 `-config` 指定其他路径）。

本地默认 `database.driver: sqlite`（`github.com/glebarez/sqlite`，**无 CGO**）。切 MySQL：

```yaml
database:
  driver: mysql
  host: 127.0.0.1
  port: 3306
  user: root
  password: "your-password"
  name: chengzi_agent
```

## 统一响应信封

业务接口 JSON 固定为：

```json
{ "code": 0, "msg": "", "data": null, "sysMsg": "" }
```

- `code`：`0` 成功，`1` 失败
- HTTP 默认 **200**，由 `code` 区分业务成败（方案 A）

实现见 `pkg/response`。

## 手写依赖注入

装配顺序在 `internal/app/app.go`：

`Config` → `DB` → `Repository` → `Service` → `Handler` → `Router`

新增业务时按同样顺序扩展构造函数注入，不引入 wire/dig。

## 打包

```bash
make build
# 或 ./build.sh
```

产物在 `build/`，命名：`chengzi-agent-server-{goos}-{arch}`。

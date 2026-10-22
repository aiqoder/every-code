# ADE · Agent Development Environment

用一句话描述需求，剩下的交给 Agent：拆解待办、在独立 worktree 里写代码、跑验收、合入 dev、复核后合入 main、发布到远程主机。

- **`server/`**：Go 服务端（Gin + GORM + SQLite），持有项目、流水线、智能体调度、技能库与部署档案。
- **`desktop/`**：Electron 桌面端（Vue 3 + Pinia + xterm），承载四个工位与公共 Shell，本机执行 Git / 文件 / 命令 / SSH。

两者通过 WebSocket 执行面（`desktop/src/main/chengzi/link.ts` ↔ `server/pkg/clienthub`）连接。服务端只下发「调用哪个工具、在哪个目录」，真正碰文件的永远是本机。

## 1. 启动

### 服务端

```bash
cd server
go run ./cmd/server -config etc/ade-dev.yaml
```

监听 `127.0.0.1:8183`，数据库落在 `server/data/app.db`。

`etc/ade-dev.yaml` 是本机联调配置，与 `etc/ade.yaml` 的唯一差别是打开了 `DevAuth`：

```yaml
DevAuth:
  Enabled: true
  UserID: 1
  Username: 本地开发者
  Token: ade-dev-token
```

打开后 `/api/auth/login` 不访问 user-feedback，直接发固定令牌，省去登记 OAuth 客户端。**部署到共享环境前必须改用 `etc/ade.yaml`，并保持 `DevAuth.Enabled=false`。**

Redis 是可选依赖：连不上时自动退回进程内存（日志会提示），代价是重启后要等桌面端心跳重新上报 catalog（约 30–60 秒），期间技能顾问与拆解不可用。

### 桌面端

```bash
cd desktop
npm install
npm run dev
```

首次进入在登录页填服务地址、账号密码。本机联调用任意用户名密码即可（DevAuth 放行）。

其它命令：

```bash
npm run typecheck   # vue-tsc，渲染层
npx tsc --noEmit -p tsconfig.node.json   # 主进程与 preload
npm run build       # 三段打包到 out/
npm run dist:mac    # 打 dmg
```

## 2. 四个工位

侧栏是「项目 + 四工位」，底部是四工位共用的多标签 Shell，切页不销毁。

### 创建

- 选场景（官网 / 后台 / 小程序 …）或直接勾选技术栈；拿不准就点「让 Agent 推荐」，`ade-advisor` 会给出技术与理由。
- 可选「检测开发环境」：逐项在本机跑 `go version`、`node -v` 之类，结果回写项目。
- 提交任务描述后由 `ade-planner` 拆解，两种模式二选一：
  - **立即拆解**：拆完停下等人点「开始执行」。
  - **自动执行**：拆完直接进入开发。

### 执行

左栏是六状态任务列：待执行 / 执行中 / 待测试 / 测试完成 / 重新执行 / 执行出错。每条任务对应一个独立 git worktree：

```
<repo>                                          主仓库，dev 与 main 在这里
~/.every-code/worktrees/<项目目录名>/<任务id>     该任务的 worktree，分支 ade/<id>
```

所有任务 worktree 统一放在用户主目录下的 `.every-code/worktrees/`，按项目分子目录。

任务详情页维护名称、概览与润色描述。Agent 生成的待办锁定不可改；用户追加的待办可以改、可以删。

### 测试

验收命令在 worktree 里跑，通过后合入 `dev`，再由 `ade-reviewer` 复核，复核通过才合入 `main`。失败可以要求重新执行，重试超过 3 次置为执行出错。

### 部署

- 主机档案与发布剧本存服务端；**私钥只在本机**（`safeStorage` 加密存密钥正文，不写服务端数据库）。
- 首次连接会回报主机指纹，核对后回写服务端，后续连接按指纹校验。
- 剧本是极简 YAML 子集，步骤分三类：`local`（本机构建）、`upload`（SFTP 上传）、`remote`（远程命令），可用 `{{version}}` `{{previous}}` `{{remoteDir}}` `{{taskId}}` `{{branch}}` `{{host}}` 占位。
- 远程布局约定：`<remoteDir>/releases/<时间戳>`、`<remoteDir>/current` 软链、`<remoteDir>/shared`。
- 失败按 `rollback` 把 `current` 指回上一版；每条远程命令都写审计（退出码 + 输出），与发布记录一起可查。

## 3. 五个智能体

| id | 职责 |
|----|------|
| `ade-advisor` | 推荐技术栈与场景 |
| `ade-planner` | 把需求拆成待办，用工具写回任务 |
| `ade-coder` | 在 worktree 里逐条实现并提交 |
| `ade-verifier` | 跑验收命令，判定通过与否 |
| `ade-reviewer` | 复核 diff，决定能否合入 main |

提示词与工具集都在桌面端 `desktop/src/main/chengzi/link.ts` 里，随 catalog 上报给服务端。工具的落脚目录由服务端随 `tool_call` 下发（会话绑定的 worktree 优先），主进程再用 `resolveInWorkspace` 兜住越界路径。

开发智能体用 read_context 读取计划，并用本机 shell 执行 `git worktree add` 创建任务目录。

## 4. 技能

系统技能在 `server/skills/<id>/SKILL.md`，目前有 7 个：`go-service`、`vue-app`、`php-service`、`java-service`、`node-service`、`html-app`、`wechat-miniprogram`。

接口只做三件事，客户端不提供在线编辑：

- `GET /api/ade/skills` 列表，`GET /api/ade/skills/:id/files` 文件树，`GET /api/ade/skills/:id/file?path=` 单文件正文
- `GET /api/ade/skills/:id/download` 下载整包 zip
- `POST /api/ade/skills/:id/upload` 上传 zip，**整包替换**

用户自定义技能另存本机 `~/Library/Application Support/ade-desktop/skills/<id>/SKILL.md`，同 id 会盖过系统技能，只在这台机器生效。

## 5. 安全基线

- 渲染进程 `contextIsolation: true`、`nodeIntegration: false`；所有 REST 请求走主进程，渲染层拿不到 token。
- token 用 Electron `safeStorage` 加密后存 `settings.json`，不可用时降级为 base64 并明确标注。
- 服务器配置里没有生产密钥；OAuth `ClientSecret` 留空由部署方填。
- 私钥不出本机：选择私钥文件后，正文用 Electron `safeStorage` 加密，按主机档案 id 存本机；部署前必须由人点「确认发布」，远程命令全部落审计并可在部署页查看。

## 6. 目录结构

```
every-code/
├── server/                     Go 服务端
│   ├── cmd/server/             入口
│   ├── etc/                    ade.yaml / ade-dev.yaml
│   ├── internal/
│   │   ├── config/             配置结构与校验
│   │   ├── handler/ade/        /api/ade 全部处理器
│   │   ├── logic/ade/          事件、服务、拆解、执行、调度、技能、部署
│   │   ├── model/ repository/  数据模型与访问
│   │   └── svc/                依赖装配
│   ├── pkg/
│   │   ├── agentloop/          通用智能体循环
│   │   ├── clienthub/          WebSocket 执行面（含 /rpc 白名单）
│   │   └── catalogstore/       Redis + 内存双实现
│   └── skills/                 系统技能
└── desktop/                    Electron 桌面端
    ├── src/main/               主进程：store/server/git/local/ssh/pty/chengzi/sse
    ├── src/preload/            contextBridge 暴露的 window.ade
    └── src/renderer/           Vue 3 渲染层（views / stores / components）
```

## 7. 已知限制

- 部署需要一台可 SSH 公钥登录的主机；服务端不保存私钥，换机器要重新指定。
- catalog 走内存回退时，服务端重启后需等桌面端重新上报，advisor / planner 才可用。
- 真实登录依赖 user-feedback 后台登记的 OAuth 客户端与 `ClientSecret`。

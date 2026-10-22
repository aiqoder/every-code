# cpk 智能体安装包格式

`.cpk` 是橙子智能体的分发格式，本质为 zip 压缩包。用户通过桌面端「智能体」页本地安装、
在「商店」安装已上架应用。

## 包结构

```
agent.cpk (zip)
├── manifest.yaml          # 必需，包清单
├── prompt.md              # 系统提示词（manifest.systemPrompt 指向）
├── icon.png               # 可选头像（manifest.avatar 指向）
├── manifest.sha256        # 打包工具自动生成的逐文件 sha256 清单
├── server/                # 可选；服务端代码（打包时自动剔除，开发者自行部署）
└── skills/
    └── <skill-name>/
        ├── SKILL.md           # 必需，YAML frontmatter（name / description）
        ├── scripts/           # 桌面端脚本（入包，直接放文件，无 client 层级）
        ├── references/        # 可选，按需加载的参考文档
        └── assets/            # 可选，模板/图标等静态资源
```

## manifest.yaml

```yaml
id: weather-agent           # 必需。小写字母/数字开头，可含 - _ .，最长 64 位
name: 天气助手               # 必需
description: 查询天气的智能体  # 可选，≤512 字
version: 1.0.0              # 必需，语义化版本
author: someone             # 可选
minClientVersion: 0.1.0     # 可选，要求桌面端最低版本
systemPrompt: prompt.md     # 必需，提示词文件（相对包根目录）
avatar: icon.png            # 可选，头像文件（宽高均 ≤256 像素）
maxSteps: 0                 # 可选，单轮最大模型调用步数（0=服务端默认，上限 999）
skills: [weather]           # 可选，显式声明技能目录；省略则扫描 skills/ 下全部
server:
  baseUrl: https://api.example.com  # 可选，开发者服务器地址（注入为 CHENGZI_SERVER_BASE_URL）
env:                            # 可选。开发者声明变量名，用户在桌面端填写取值
  - name: API_TOKEN             # 环境变量名；脚本按此名称读取
    label: 接口令牌              # 可选，展示给用户的说明
    secret: true                # 可选，按密文输入，默认 false（仅 type=string）
    required: true              # 可选，未填写则不能保存，默认 false
  - name: TENANT_ID
    label: 租户编号
  - name: BT_PANELS             # type=list：结构化清单，桌面端按行填写
    label: 面板清单
    required: true
    type: list
    items:
      - name: alias
        label: 别名
      - name: host
        label: 主机
        required: true
      - name: port
        label: 端口
      - name: user
        label: 用户
      - name: password
        label: 密码
        secret: true
```

约束：

- `env.name`：字母或下划线开头，仅含字母、数字、下划线，最长 64 位；不能使用 `CHENGZI_` 前缀；大小写不同也视为重复
- `env.label` 可选，≤64 字；`secret`、`required` 默认 false
- `env.type`：`string`（默认，单行输入）或 `list`（可增删行的结构化清单）
- `type: list` 时必须声明 `items`（1–16 列）；每列有 `name`（命名规则同 env.name）、可选 `label` / `secret` / `required`；顶层 `secret` 无效，密文由列级 `secret` 控制
- `list` 注入值：按 `items` 顺序用 ` | ` 拼接各列，去掉末尾空字段，多行用换行分隔；脚本按字符串读取即可
- 变量取值不写进安装包，由用户安装后在桌面端填写
- `systemPrompt` / `avatar` 不允许绝对路径与 `..` 穿越
- 包内所有技能目录必须包含 `SKILL.md`

## 安装行为（桌面端）

1. 校验 manifest 与包结构、桌面端版本门槛（`minClientVersion`）
2. 解包到 `~/.chengzi-agent/packages/<id>/<version>/`，索引写入 `packages.json`
3. 包内技能注册为 `pkg:<包id>:<技能目录名>` 形式的客户端技能，上报服务端
4. 经 `POST /api/agents` 注册为当前用户的私有智能体（名称/简介/提示词/技能/maxSteps/版本）
5. 升级：同 id 更高版本重复安装即升级（不允许回退到低版本），同步 `PUT /api/agents/:id`
6. 卸载：删除服务端智能体 + 本地全部文件与用户填写的环境变量

## 脚本与服务端约定

不要求特定 SDK 或语言。`skills/<技能名>/scripts/` 与包根 `server/`（开发者自行部署）只要遵守下面的环境变量约定即可（Node、Python、PHP、Shell、Go 等均可）。

agent 命中技能后，按 SKILL.md 指引经桌面端 `shell_exec` 运行 `scripts/` 内脚本。命令引用包目录下路径时自动注入：

- `CHENGZI_SERVER_BASE_URL`：manifest 的 `server.baseUrl`（有则注入）
- `CHENGZI_PACKAGE_ID` / `CHENGZI_PACKAGE_DIR`
- `env` 中已声明、且用户已填写的环境变量。变量名由开发者决定，取值由用户在桌面端配置，只保存在本机 `~/.chengzi-agent/packages/<id>/env.json`（权限 0600）

脚本直接读这些环境变量，自行决定怎么发给自己的服务。平台不代为登录，也不规定账号密码、令牌文件或请求格式。未填写的变量不会注入。

## 打包工具

在本仓库开发时：

```bash
go run ./cmd/cpkpack -src ./my-agent -out ./my-agent.cpk
```

「软件开发助手」安装包（`examples/cpk/software-dev-agent`）内置同一套规则，脚本在技能目录里。桌面端安装后执行：

```bash
node <技能目录>/scripts/cpkpack.js -src ./my-agent -out .
```

脚本按 manifest 把文件名固定为 `<id>-<version>.cpk`，`-out` 只指定目录。例如 `id: weather-agent`、`version: 1.0.0` 时生成 `weather-agent-1.0.0.cpk`。

两种入口都会剔除包根 `server/` 目录并生成 `manifest.sha256`。

## 应用商店

平台智能体不再通过管理端表单填写提示词和技能，统一以 `.cpk` 上架：

- 管理员在「应用商店」上传 `.cpk` 后直接上架（`source=official`）
- 客户端用户上传 `.cpk` 申请上架，状态为 `pending`，管理员审核通过后变为 `published`
- 同一 `manifest.id` 的新版本必须高于当前已上架版本；上架后更低版本自动下架
- 应用标识已被他人占用（待审核或已上架）时，其他用户不能再提交；管理员上传可以占用该标识
- 客户端「商店」只展示已上架应用，安装走既有的本地解包与私有智能体注册流程

接口：

- `GET /api/store/apps` 已上架列表
- `GET /api/store/mine` 我的申请
- `POST /api/store/apps` multipart 字段 `file`，申请上架
- `GET /api/store/apps/:id/download` 下载已上架安装包
- `DELETE /api/store/apps/:id` 撤回待审核或已驳回的申请
- `GET /api/admin/store/apps?status=` 管理列表
- `POST /api/admin/store/apps` 管理员直接上架
- `POST /api/admin/store/apps/:id/review` 请求体 `{"action":"approve|reject","note":""}`
- `POST /api/admin/store/apps/:id/offline` 下架
- `DELETE /api/admin/store/apps/:id` 删除记录与文件

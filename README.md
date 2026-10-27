# Every Code

源码：https://github.com/aiqoder/every-code

本项目以 [MIT](LICENSE) 许可证开源。

## 工具

本机不自带读文件、改代码或执行命令的工具。这些能力放在编程助手插件 `coding-agent` 的 `tools/` 里，安装后才会注册。用户也可以把同样格式的 `.js` 直接放进 `~/.every-code/tools/`。

## 自定义工具

工具只能用 `.js`，一个文件定义一个工具，格式固定为 CommonJS：

```js
module.exports = {
  name: 'echo',
  description: '回显文本',
  parameters: [
    { name: 'text', type: 'string', description: '要回显的内容', required: true }
  ],
  async run(args, ctx) {
    return String(args.text ?? '')
  }
}
```

- `name` 以小写字母开头，只含小写字母、数字和下划线。
- `parameters[].type` 只能是 `string`、`number`、`integer`、`boolean`。
- `run` 返回字符串，超时 150 秒。`ctx.workspace` 是当前工作目录，`ctx.env` 是插件环境变量，`ctx.agentId` 是本次调用的智能体编号，`ctx.packageDir` 是插件目录。
- 可以 `require` Node 内置模块。不要依赖额外的 npm 包。
- 文件直接放在 `~/.every-code/tools/`。子目录、`.ts` 和其他后缀不会加载。

插件包里的 `tools/*.js` 使用同一格式，安装后和用户目录里的工具一起注册。同名时保留用户目录里的定义。

## 智能体插件

顶栏「智能体」页可以安装 `.cpk` 或 `.agent`。两者都是 zip，清单与 chengzi 的 `.cpk` 相同，并可以带上 `tools/*.js`：

```
manifest.yaml
prompt.md
skills/<name>/SKILL.md
tools/<name>.js
```

安装到 `~/.every-code/packages/`。登录后会把提示词、技能和工具同步到服务器。技能和工具都走 websocket，由这台电脑读取技能目录并执行 `.js` 工具。

## Recommended IDE Setup

- [VSCode](https://code.visualstudio.com/) + [ESLint](https://marketplace.visualstudio.com/items?itemName=dbaeumer.vscode-eslint) + [Prettier](https://marketplace.visualstudio.com/items?itemName=esbenp.prettier-vscode) + [Volar](https://marketplace.visualstudio.com/items?itemName=Vue.volar)

## Project Setup

### Install

```bash
$ npm install
```

### Development

```bash
$ npm run dev
```

### 环境变量

| 变量                | 说明                                                         |
| ------------------- | ------------------------------------------------------------ |
| `VITE_API_BASE_URL` | 后端 API 基地址，供 `src/renderer/src/utils/request.ts` 使用 |
| `VITE_APP_TITLE`    | 顶栏标题                                                     |

开发与生产示例分别在 `.env.development`、`.env.production`。键名清单见 `.env.example`。含密钥的值放 `.env.development.local` / `.env.production.local`，不要提交。

### 渲染进程目录

```
src/renderer/src/
├── api/            # 接口
├── stores/         # Pinia
├── utils/request.ts
├── components/     # 全局基础组件
├── layouts/
├── views/          # 页面，业务组件放 views/<模块>/components/
└── router/
```

### Build

```bash
# For windows
$ npm run build:win

# For macOS
$ npm run build:mac

# For Linux
$ npm run build:linux
```

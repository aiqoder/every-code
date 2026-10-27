# Every Code

Electron + Vue 3 + TypeScript 桌面端（electron-vite）。

源码：https://github.com/aiqoder/every-code

本项目以 [MIT](LICENSE) 许可证开源。

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

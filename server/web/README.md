# 管理后台（Vue 3）

橙子 Agent 管理端，技术栈：Vue 3 + Vite + TypeScript + Pinia + Vue Router + Element Plus + Tailwind CSS。

## 目录

```
web/
├── src/
│   ├── api/              # 接口封装
│   ├── assets/
│   ├── layouts/          # 后台布局
│   ├── router/
│   ├── stores/
│   ├── types/
│   ├── utils/request.ts  # axios + 统一信封解包
│   └── views/
│       ├── dashboard/
│       └── user/
├── .env.development
├── .env.production
└── .env.example
```

## 启动

先启动 Go API（默认 `8182`），再启动前端：

```bash
cd web
npm install
npm run dev
```

开发代理：`/api` → `http://127.0.0.1:8182`。

## 环境变量

| 变量 | 说明 |
|------|------|
| `VITE_API_BASE_URL` | API 基地址，开发默认 `/api` |
| `VITE_APP_TITLE` | 页面标题 |

## 接口契约

与后端 `pkg/response` 一致：`{ code, msg, data, sysMsg }`，`code === 0` 成功。前端在 `utils/request.ts` 中解包 `data`，失败时提示 `msg` 并 `console.error(sysMsg)`。

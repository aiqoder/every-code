---
name: Node.js 服务
description: 用 Node.js 写接口、脚本或构建任务时使用。约束模块规范、异步错误处理、依赖管理与进程退出，避免写出挂住不动的服务。
---

# Node.js 服务

## 工程形态

- 新项目 `npm init -y` 后改 `"type": "module"`，统一用 ESM。
- 入口 `src/index.js` 或 `src/server.js`，只做装配与监听。
- 路由、业务、数据访问分目录；不把所有东西塞一个文件。
- `package.json` 里写清 `engines.node`，锁文件进仓库。

## 必守规则

1. **异步错误必须接住。** `await` 外面要有 try/catch，或用 `process.on('unhandledRejection')` 兜底并记日志。
2. **不要在请求处理里同步读大文件。** 用 `fs/promises`。
3. **端口、密钥从 `process.env` 读。** 不写死，也不把 `.env` 提交。
4. **信号要处理。** `SIGINT` / `SIGTERM` 时关掉 server 与数据库连接再退出。
5. **依赖要克制。** 一个功能装一个大包之前，先看标准库够不够。

## 常写片段

```js
import http from 'node:http'

const server = http.createServer(async (req, res) => {
  try {
    const data = await handle(req)
    res.writeHead(200, { 'Content-Type': 'application/json; charset=utf-8' })
    res.end(JSON.stringify({ code: 0, data }))
  } catch (err) {
    console.error('request failed', err)
    res.writeHead(500, { 'Content-Type': 'application/json; charset=utf-8' })
    res.end(JSON.stringify({ code: 1, msg: '服务异常' }))
  }
})

const shutdown = () => server.close(() => process.exit(0))
process.on('SIGINT', shutdown)
process.on('SIGTERM', shutdown)

server.listen(Number(process.env.PORT) || 3000)
```

## 验收

- `node --check` 或 `npm run lint` 通过。
- 服务能起来，`curl` 主要接口有正常返回。
- 触发一次异常路径，进程不崩、日志有记录。

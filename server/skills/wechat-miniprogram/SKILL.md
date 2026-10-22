---
name: 微信小程序
description: 开发微信小程序时使用。约束目录结构、页面生命周期、分包与接口调用，避免出现真机上跑不起来的工程。
---

# 微信小程序

## 工程形态

```text
project.config.json
app.json
app.js
app.wxss
pages/
  index/
    index.wxml
    index.wxss
    index.js
    index.json
components/        自定义组件
utils/             工具函数
```

- `project.config.json` 里 `appid` 用占位值 `touristappid` 时只能用测试号，不要写死他人 appid。
- 每个页面四件套齐全，`app.json` 的 `pages` 数组里注册。
- 组件用 `"component": true`，页面用 `Page({})`，不要混。

## 页面规范

1. **`onLoad` 只取参数，`onShow` 做刷新。** 不要在两个钩子里重复请求。
2. **请求封装。** `wx.request` 统一包一层，处理域名白名单、超时与错误码，不要每个页面各写一遍。
3. **状态用 `setData`。** 不要直接改 `this.data`。
4. **列表加 `wx:key`。** 用业务 id，不要用 index。
5. **图片控制大小。** 大图放 CDN 或用 `image` 的 `mode` 裁切，避免包体超限。

## 常写片段

```js
// utils/request.js
export function request(path, { method = 'GET', data } = {}) {
  return new Promise((resolve, reject) => {
    wx.request({
      url: `${getApp().globalData.baseUrl}${path}`,
      method,
      data,
      success(res) {
        if (res.statusCode === 200 && res.data && res.data.code === 0) resolve(res.data.data)
        else reject(new Error((res.data && res.data.msg) || '请求失败'))
      },
      fail(err) { reject(new Error(err.errMsg || '网络异常')) },
    })
  })
}
```

## 验收

- 微信开发者工具能编译通过，无红色报错。
- 每个页面能打开，`app.json` 里注册的路径都存在。
- 请求走的是合法域名，或用「不校验合法域名」在本地调试通过。

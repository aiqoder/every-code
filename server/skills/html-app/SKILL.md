---
name: 原生 HTML
description: 不引入构建链，直接写 HTML/CSS/JavaScript 页面时使用。约束语义化结构、响应式布局与无框架交互写法，做出打开即用的页面。
---

# 原生 HTML

## 文件形态

```text
index.html        页面结构
styles.css        样式
app.js            交互
```

- 直接双击或静态服务器打开就能用，不需要编译。
- 不引入构建工具；需要第三方库时用 CDN 的固定版本，不用 `latest`。

## 页面规范

1. **语义化标签。** 用 `header` / `nav` / `main` / `section` / `footer`，不要全是 `div`。
2. **语言与字符集。** `<html lang="zh-CN">`、`<meta charset="UTF-8">`。
3. **移动端可用。** 加 `<meta name="viewport" content="width=device-width, initial-scale=1">`，布局用 flex/grid，不用固定像素宽度撑满。
4. **可访问性。** 图片有 `alt`，表单控件有 `label`，按钮用 `button` 不用可点的 `div`。
5. **不写内联事件。** `addEventListener` 绑定，脚本放 `</body>` 前或加 `defer`。

## 常写片段

```html
<!doctype html>
<html lang="zh-CN">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>页面标题</title>
  <link rel="stylesheet" href="styles.css">
</head>
<body>
  <main id="app"></main>
  <script src="app.js" defer></script>
</body>
</html>
```

```js
// 渲染列表并处理点击
const app = document.querySelector('#app')
app.addEventListener('click', (e) => {
  const btn = e.target.closest('[data-action]')
  if (!btn) return
  // ...
})
```

## 验收

- 浏览器打开无控制台报错。
- 桌面与手机宽度下布局都不错位。
- 所有交互点一遍有反应。

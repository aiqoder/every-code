---
name: Vue 应用
description: 用 Vue 3 写前端页面、组件或后台时使用。约束目录、组件拆分、状态管理与交互细节，做出可直接交付的界面而不是一堆占位块。
---

# Vue 应用

## 工程形态

优先 `npm create vue@latest` 或 `pnpm create vue` 生成，不要手搓构建链。

```text
src/
  main.ts
  App.vue
  router/index.ts
  stores/             Pinia，一个领域一个 store
  views/              路由页面
  components/         可复用组件，按领域分子目录
  api/                接口封装，一函数一接口
  assets/
```

## 组合式 API 规范

- 一律 `<script setup lang="ts">`。
- props 用 `defineProps<{...}>()`，事件用 `defineEmits<{...}>()`，两边都写类型。
- 不用 `any`；接口返回类型放在 `src/api/types.ts`。
- 跨组件共享的状态进 Pinia，页面局部状态留在组件里。

## 交互细节

1. **每个列表都要有空态。** 没有数据时给一句人话提示和下一步动作，不要留白。
2. **每个异步动作都要有加载与失败反馈。** 按钮在请求期间禁用，失败给出可读原因。
3. **不要用 `alert`。** 用页面内的提示条或轻量弹层。
4. **表单要校验。** 必填项为空时在字段旁提示，不要等提交后才报错。
5. **路由页面有标题。** `router.afterEach` 里设 `document.title`。

## 样式

- 优先用项目现有的样式方案（Tailwind / scoped CSS / 组件库），不要引入第二套。
- 颜色、间距、字号集中定义，避免散落的魔法值。
- 尊重 `prefers-reduced-motion`，动效不要过头。

## 验收

- `npm run build` 或 `pnpm build` 通过，无类型错误。
- 打开页面能看到真实数据或完整空态，控制台无红色报错。
- 关键交互（提交、删除、翻页）手动点一遍可用。

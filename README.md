> 本项目依托橙子ai平台而开发，软件并不负责agent推理，所有推理都在橙子ai服务端进行，本平台会存储您的数据以提高服务质量，这点你必须清楚。

# Every Code 是什么

- 一个基于agent包理念的agent平台工具
- 没有任何agent引擎
- 不支持任何单独使用tools、skills
- 他还只是一个孩子

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

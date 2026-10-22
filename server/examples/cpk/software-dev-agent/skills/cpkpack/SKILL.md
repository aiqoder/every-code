---
name: cpkpack
description: 把智能体源目录打包为 .cpk。用户说「打包」「打成 cpk」「生成安装包」时使用。
---

# cpkpack

把已经写好的包源目录打成 `.cpk`。规则与仓库里的 `cpkpack` 一致：校验 manifest、剔除包根 `server/`、生成 `manifest.sha256`。

## 何时使用

- 用户要求把当前目录或指定目录打成安装包
- 用户要校验一份包源能否被桌面端打开

编写 manifest、提示词和技能文件时，先按系统提示词里的包结构落地文件，再调用本脚本。

**每次打包前必须先升高 `manifest.yaml` 的 `version`（至少 patch +1）**，再执行本脚本。同 id 只允许安装更高版本。

## 命令

技能目录见绑定技能段，或本文件加载时前置的目录提示。必须用绝对路径：

```bash
node <技能目录>/scripts/cpkpack.js -src <包源目录> -out <输出目录>
```

- `-src`：含 `manifest.yaml` 的包源目录
- `-out`：输出目录；不存在时会创建
- 文件名固定为 `<id>-<version>.cpk`，不能自定义
- 成功时退出码 0，stdout 打印输出路径、id、name、version、skills、files
- 参数缺失退出码 2，打包失败退出码 1，原因在 stderr

## 打包时会做的事

- 读取并校验 `manifest.yaml`（id、name、version、systemPrompt、env.name / type / items 与 server.baseUrl）
- 确认 `systemPrompt` 和可选 `avatar` 指向的文件存在，且是包内相对路径；`avatar` 宽高均不得超过 256 像素
- 跳过以 `.` 开头的文件和目录（如 `.venv`、`.git`）、`__MACOSX`、`node_modules`、`venv`、`__pycache__`、已有的 `manifest.sha256` 与 `*.cpk`
- 跳过包根 `server/` 整棵目录（服务端代码不入包）
- 对其余文件按路径排序，写逐文件 sha256，清单本身不参与校验
- 打成 zip，并写入 `manifest.sha256`

不要改这个脚本去迁就某一份包。源目录不符合约定时，把 stderr 原文告诉用户并改源文件。

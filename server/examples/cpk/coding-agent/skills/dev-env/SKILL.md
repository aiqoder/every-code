---
name: dev-env
description: >
  搭建/安装开发环境：Python（pyenv）、Node.js（nvm）、Java（SDKMAN/包管理器）、Go、Rust 等运行时与版本管理。
  用户说「安装 Python/Node/Java/Go」「配开发环境」「装 pyenv/nvm」「切换国内源装运行时」「初始化工具链」时使用。
  能区分中国大陆与海外网络，国内优先镜像加速安装并配置包管理器源。
---

# 开发环境安装与版本管理

按目标主机的网络归属选择下载源，快速安装常用语言运行时与版本管理器。**包管理器换源细节**见 `mirror-source` 技能；本技能聚焦**运行时本体**的安装与切换。

## 使用时机

- 机器缺少 Python / Node.js / Java / Go / Rust 等
- 需要安装或升级 pyenv、nvm、SDKMAN
- 需要切换多版本（如 Python 3.11 / 3.12，Node 18 / 20）
- 国内网络下官方下载慢或失败，需镜像加速安装

## 执行流程（必须按序）

1. **识别 OS**：`uname -s` + `cat /etc/os-release`（或 macOS 用 `sw_vers`），确认架构 `uname -m`。
2. **判断网络**：依据系统提示词「目标环境」公网 IP 归属；不确定时探测：
   ```bash
   # 能访问国内镜像 → 视为大陆网络；两侧都通则按 IP 归属，优先国内镜像更稳
   curl -sI --max-time 5 https://mirrors.tuna.tsinghua.edu.cn | head -1
   curl -sI --max-time 5 https://www.python.org | head -1
   ```
3. **检查是否已装**：`command -v python3/node/java/go` 与版本管理器 `pyenv/nvm/sdkman`。已有且版本满足需求则跳过安装，只补换源。
4. **按下文安装**：大陆用「国内」命令块，海外用「官方」命令块。
5. **安装后换包源**：pip / npm / GOPROXY / Maven 等按 `mirror-source` 配置。
6. **验证**：打印版本号，确认 `which` 指向版本管理器路径而非过期系统包。

### 网络策略速查

| 场景 | 下载/安装源 | 包管理器源 |
| --- | --- | --- |
| 中国大陆 | 清华/中科大/阿里/npmmirror 等 | 国内镜像（见 mirror-source） |
| 其他地区 | 官方 GitHub / 官方发行站 | 官方默认源 |

---

## Python — 优先 pyenv

### 检查

```bash
command -v pyenv && pyenv --version
command -v python3 && python3 --version
```

### 安装 pyenv

**依赖（编译 Python 需要）**

- Debian/Ubuntu：`sudo apt update && sudo apt install -y build-essential libssl-dev zlib1g-dev libbz2-dev libreadline-dev libsqlite3-dev curl libncursesw5-dev xz-utils tk-dev libxml2-dev libxmlsec1-dev libffi-dev liblzma-dev`
- RHEL/Rocky：`sudo dnf groupinstall -y "Development Tools" && sudo dnf install -y openssl-devel bzip2-devel libffi-devel zlib-devel readline-devel sqlite-devel xz-devel`
- macOS：`xcode-select --install`；建议 `brew install openssl readline sqlite3 xz zlib tcl-tk`

**国内安装 pyenv**（GitHub 慢时用镜像 clone）：

```bash
# 探测可用 GitHub 代理后替换 <GH_PROXY>，如 https://ghproxy.net/
git clone https://<GH_PROXY>/https://github.com/pyenv/pyenv.git ~/.pyenv
# 或直接镜像（若可用）：
# git clone https://gitee.com/mirrors/pyenv.git ~/.pyenv
```

**海外**：

```bash
curl -fsSL https://pyenv.run | bash
# 或：git clone https://github.com/pyenv/pyenv.git ~/.pyenv
```

**写入 shell**（bash/zsh 按实际 shell 调整，装完后 `source` 或开新终端）：

```bash
# ~/.bashrc 或 ~/.zshrc
export PYENV_ROOT="$HOME/.pyenv"
[[ -d $PYENV_ROOT/bin ]] && export PATH="$PYENV_ROOT/bin:$PATH"
eval "$(pyenv init -)"
```

### 用 pyenv 安装 Python 版本

**国内**（加速源码下载）：

```bash
export PYTHON_BUILD_MIRROR_URL=https://mirrors.tuna.tsinghua.edu.cn/python/
# 备选：https://mirrors.huaweicloud.com/python/
pyenv install -l | grep -E '^\s*3\.(11|12|13)\.' | tail -20   # 选版本
pyenv install 3.12.8
pyenv global 3.12.8          # 或 pyenv local 3.12.8（项目目录）
python -V && which python
```

**海外**：不设 `PYTHON_BUILD_MIRROR_URL`，直接 `pyenv install <ver>`。

### pip 换源（大陆）

```bash
pip config set global.index-url https://pypi.tuna.tsinghua.edu.cn/simple
# 或：pip install -i https://pypi.tuna.tsinghua.edu.cn/simple <pkg>
```

> 无编译环境、只要快速单版本时：大陆可从 `https://mirrors.huaweicloud.com/python/` 下官方安装包；仍**优先 pyenv** 以便多版本管理。

---

## Node.js — 优先 nvm

### 检查

```bash
command -v nvm || [ -s "$HOME/.nvm/nvm.sh" ] && . "$HOME/.nvm/nvm.sh" && nvm --version
command -v node && node -v && command -v npm && npm -v
```

### 安装 nvm

**国内**：

```bash
export NVM_DIR="$HOME/.nvm"
# 用 GitHub 代理或 gitee 镜像安装脚本
curl -o- https://<GH_PROXY>/https://raw.githubusercontent.com/nvm-sh/nvm/v0.40.1/install.sh | bash
# 备选：git clone https://gitee.com/mirrors/nvm.git ~/.nvm && cd ~/.nvm && git checkout `git describe --abbrev=0 --tags`
```

**海外**：

```bash
curl -o- https://raw.githubusercontent.com/nvm-sh/nvm/v0.40.1/install.sh | bash
```

加载：

```bash
export NVM_DIR="$HOME/.nvm"
[ -s "$NVM_DIR/nvm.sh" ] && . "$NVM_DIR/nvm.sh"
```

### 用 nvm 安装 Node

**国内**（Node 二进制镜像 + npm 包镜像）：

```bash
export NVM_NODEJS_ORG_MIRROR=https://npmmirror.com/mirrors/node/
nvm install --lts
# 或：nvm install 20
nvm use --lts
nvm alias default node

npm config set registry https://registry.npmmirror.com
# pnpm / yarn 同样设 registry（见 mirror-source）
node -v && npm -v
```

将 `NVM_NODEJS_ORG_MIRROR` 写入 `~/.nvmrc` 旁的 shell profile 或 `~/.zshrc`，避免每次重装丢失：

```bash
echo 'export NVM_NODEJS_ORG_MIRROR=https://npmmirror.com/mirrors/node/' >> ~/.zshrc
```

**海外**：不设 `NVM_NODEJS_ORG_MIRROR`，`nvm install --lts` 即可。

> 已有 fnm / volta 且用户指定时，按其工具安装；默认仍用 **nvm**。fnm 国内镜像：`export FNM_NODE_DIST_MIRROR=https://npmmirror.com/mirrors/node/`。

---

## Java — SDKMAN 优先，其次包管理器 / 官方包

### 检查

```bash
command -v java && java -version
command -v javac && javac -version
[ -s "$HOME/.sdkman/bin/sdkman-init.sh" ] && echo "sdkman ok"
```

### 方式 A：SDKMAN（多版本，推荐）

**海外**：

```bash
curl -s "https://get.sdkman.io" | bash
source "$HOME/.sdkman/bin/sdkman-init.sh"
sdk install java 21.0.5-tem    # 例：Temurin 21；sdk list java 查可用版本
sdk default java 21.0.5-tem
```

**国内**：SDKMAN 官网可能慢，可：

1. 用 GitHub/代理拉取安装脚本，或
2. 直接下载 Temurin / Zulu / Dragonwell 二进制：
   - 华为云 OpenJDK：`https://mirrors.huaweicloud.com/openjdk/`
   - 阿里云：`https://mirrors.aliyun.com/java/`（路径以镜像站为准）
3. 解压到 `~/tools/jdk-21`，写入 `JAVA_HOME` 与 `PATH`。

**包管理器快速装（单版本）**：

```bash
# Debian/Ubuntu
sudo apt install -y openjdk-21-jdk
# macOS
brew install openjdk@21
```

### Maven / Gradle 换源（大陆）

见 `mirror-source`：`~/.m2/settings.xml` 阿里云 Maven；Gradle `repositories` 加阿里云。

---

## Go

### 检查

```bash
command -v go && go version
```

### 安装

先确认版本与架构，例如 `1.23.4`、`linux-amd64` / `darwin-arm64`。

**国内**（二进制镜像）：

```bash
GO_VER=1.23.4
GO_OS=$(uname -s | tr '[:upper:]' '[:lower:]')
GO_ARCH=$(uname -m | sed 's/x86_64/amd64/;s/aarch64/arm64/;s/arm64/arm64/')
curl -fsSL "https://mirrors.aliyun.com/golang/go${GO_VER}.${GO_OS}-${GO_ARCH}.tar.gz" -o /tmp/go.tgz
sudo rm -rf /usr/local/go && sudo tar -C /usr/local -xzf /tmp/go.tgz
echo 'export PATH=/usr/local/go/bin:$PATH' >> ~/.zshrc
export PATH=/usr/local/go/bin:$PATH
go version
```

**海外**：把 URL 换成 `https://go.dev/dl/go${GO_VER}.${GO_OS}-${GO_ARCH}.tar.gz`。

**macOS Homebrew**：`brew install go`（大陆先按 mirror-source 配 Homebrew）。

### 模块代理（大陆，装完必配）

```bash
go env -w GOPROXY=https://goproxy.cn,direct
go env -w GOSUMDB=sum.golang.google.cn
```

---

## Rust（可选）

**国内**：

```bash
export RUSTUP_DIST_SERVER=https://mirrors.ustc.edu.cn/rust-static
export RUSTUP_UPDATE_ROOT=https://mirrors.ustc.edu.cn/rust-static/rustup
curl --proto '=https' --tlsv1.2 -sSf https://mirrors.ustc.edu.cn/rust-static/rustup/dist/$(uname -m)-unknown-linux-gnu/rustup-init -o rustup-init
# macOS/其他平台用对应 rustup-init，或：
curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh
```

写入 Cargo 镜像见 `mirror-source`。

**海外**：`curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh`

---

## 一键探测：已装工具清单

安装前后可跑，便于向用户汇报：

```bash
echo "=== OS ===" && uname -srm && (cat /etc/os-release 2>/dev/null | head -3 || sw_vers 2>/dev/null)
echo "=== Python ===" && (pyenv --version 2>/dev/null; python3 --version 2>/dev/null; which python3 2>/dev/null)
echo "=== Node ===" && ( [ -s "$HOME/.nvm/nvm.sh" ] && . "$HOME/.nvm/nvm.sh"; nvm --version 2>/dev/null; node -v 2>/dev/null; npm -v 2>/dev/null)
echo "=== Java ===" && (java -version 2>&1 | head -1; echo "JAVA_HOME=$JAVA_HOME")
echo "=== Go ===" && (go version 2>/dev/null; go env GOPROXY 2>/dev/null)
echo "=== Rust ===" && (rustc --version 2>/dev/null)
```

---

## 注意事项

1. **版本管理器优先于系统包**：Python → pyenv，Node → nvm；避免 `apt install python3` / `apt install nodejs` 造成版本过旧且难切换。
2. **权限**：pyenv/nvm 装用户目录，一般不需要 sudo；Go 解压到 `/usr/local` 需要 sudo，也可改装 `$HOME/go-sdk`。
3. **GitHub 不可达**：安装 pyenv/nvm 前先按 `mirror-source` 探测 GitHub 代理，再替换 clone/curl URL。
4. **私有凭据**：不要把 token 写进镜像代理 URL。
5. **与 mirror-source 分工**：本技能装运行时；pip/npm/apt/docker 等日常换源交给 `mirror-source`。
6. **验证再结束**：每个工具安装后必须跑版本命令；失败时检查镜像 URL、架构、编译依赖是否齐全。

## 输出给用户

完成后简要说明：

- 判定的网络环境（大陆 / 海外）
- 安装了哪些工具及版本
- 写入了哪些环境变量 / shell 配置（需用户 `source` 或重开终端时明确提示）
- 已配置的包源（若有）

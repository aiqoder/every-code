---
name: mirror-source
description: apt/yum/dnf/apk/pacman/opkg 系统源，pip/uv/npm/pnpm/yarn/go/cargo/rustup/maven/gradle/composer/gem/conda 等语言包管理器，docker/containerd 拉取镜像，编写 Dockerfile（RUN apt/pip/npm 换源），GitHub clone/release/raw、Hugging Face 模型、Homebrew等下载失败问题
---
# 镜像源与下载加速

依据系统提示词中「目标环境」的公网 IP 归属判断网络环境。**中国大陆**主机优先用下列国内镜像；其他地区用官方源或就近镜像。给出安装/下载命令时应显式采用合适的源，避免默认源不可达导致失败。

## 使用原则

1. **先识别 OS 与包管理器**：`cat /etc/os-release`，确认是 apt / dnf / yum / apk / pacman / opkg 中的哪一种，再选对应章节。
2. **改源前备份**：`cp` 原配置文件为 `.bak`，改完立即 `update` / `makecache` 验证。
3. **优先官方镜像站**：高校 TUNA/USTC、云厂商阿里/腾讯/华为；社区 Docker/GitHub 代理仅作补充。
4. **部署前探测**：公共代理可用性变化快，用文末探测脚本或 `curl -I --max-time 10 <url>` 确认后再写入配置。
5. **私有凭据不走第三方代理**：GitHub 私仓、带 token 的请求、企业内网 registry 禁止走公益反代。

### 国内镜像站总览（按场景选域名）

| 镜像站 | 域名 | 擅长 |
| --- | --- | --- |
| 清华 TUNA | `mirrors.tuna.tsinghua.edu.cn` | Debian/Ubuntu、Docker CE、Rust、Conda、Helm、高校首选 |
| 中科大 USTC | `mirrors.ustc.edu.cn` | Debian/Ubuntu、Alpine、Docker CE、Rust、OpenWrt |
| 阿里云 | `mirrors.aliyun.com` | Rocky/Alma、Maven、Go 二进制、Composer |
| 腾讯云 | `mirrors.tencent.com` / `mirrors.cloud.tencent.com` | 系统源、Maven、npm |
| 华为云 | `mirrors.huaweicloud.com` | 系统源、PyPI、Maven、Python 安装包 |
| 网易 | `mirrors.163.com` | 系统源（Debian/Ubuntu/CentOS/Alpine） |
| 北外 BFSU | `mirrors.bfsu.edu.cn` | 系统源 |
| 上海交大 SJTUG | `mirror.sjtu.edu.cn` | Arch、Docker Hub（高校） |
| 南京大学 NJU | `mirrors.nju.edu.cn` | 系统源、Docker Hub（高校） |

> 同一镜像站通常覆盖多种发行版，路径见各站 `/help/` 页面。下文以 **USTC** 或 **TUNA** 为例，可互换为表中其它站（保持路径结构一致）。

---

## 操作系统软件源

### apt — Debian / Ubuntu / 衍生版

**适用**：Debian、Ubuntu、Raspberry Pi OS、多数 LXC 容器。

**配置文件**（先确认本机实际文件）：
- 传统：`/etc/apt/sources.list`
- DEB822（Debian 12+、Ubuntu 24.04+）：`/etc/apt/sources.list.d/debian.sources` 或 `ubuntu.sources`

**Debian 12（bookworm）DEB822 示例**（`MIRROR` 换 `mirrors.ustc.edu.cn` 或 `mirrors.tuna.tsinghua.edu.cn`）：

```bash
sudo cp /etc/apt/sources.list.d/debian.sources /etc/apt/sources.list.d/debian.sources.bak 2>/dev/null || true
sudo tee /etc/apt/sources.list.d/debian.sources >/dev/null <<'EOF'
Types: deb
URIs: https://mirrors.ustc.edu.cn/debian
Suites: bookworm bookworm-updates bookworm-backports
Components: main contrib non-free non-free-firmware
Signed-By: /usr/share/keyrings/debian-archive-keyring.gpg

Types: deb
URIs: https://mirrors.ustc.edu.cn/debian-security
Suites: bookworm-security
Components: main contrib non-free non-free-firmware
Signed-By: /usr/share/keyrings/debian-archive-keyring.gpg
EOF
sudo apt update
```

**Ubuntu 24.04（noble）DEB822 示例**：

```bash
sudo sed -i.bak \
  's|http://archive.ubuntu.com|https://mirrors.tuna.tsinghua.edu.cn|g; s|http://security.ubuntu.com|https://mirrors.tuna.tsinghua.edu.cn|g' \
  /etc/apt/sources.list.d/ubuntu.sources
sudo apt update
```

**一键替换（传统 sources.list）**：

```bash
sudo sed -i.bak \
  -e 's|deb.debian.org|mirrors.ustc.edu.cn|g' \
  -e 's|security.debian.org|mirrors.ustc.edu.cn/debian-security|g' \
  -e 's|archive.ubuntu.com|mirrors.tuna.tsinghua.edu.cn|g' \
  -e 's|security.ubuntu.com|mirrors.tuna.tsinghua.edu.cn|g' \
  /etc/apt/sources.list
sudo apt update
```

**Proxmox VE**：除 Debian 基础源外，还需改 PVE 源 → `https://mirrors.ustc.edu.cn/proxmox/debian/pve`。

---

### dnf / yum — Rocky / Alma / CentOS Stream / Fedora / RHEL 系

**适用**：Rocky Linux、AlmaLinux、CentOS Stream、Fedora、Oracle Linux 等。

**Rocky / Alma（禁用 mirrorlist，改用阿里云 baseurl）**：

```bash
sudo cp -a /etc/yum.repos.d /etc/yum.repos.d.bak
# Rocky
sudo sed -i.bak \
  -e 's|^mirrorlist=|#mirrorlist=|g' \
  -e 's|^#baseurl=https://dl.rockylinux.org/\$contentdir|baseurl=https://mirrors.aliyun.com/rockylinux|g' \
  /etc/yum.repos.d/Rocky-*.repo
# Alma
sudo sed -i.bak \
  -e 's|^mirrorlist=|#mirrorlist=|g' \
  -e 's|^# baseurl=https://repo.almalinux.org|baseurl=https://mirrors.aliyun.com/almalinux|g' \
  /etc/yum.repos.d/almalinux*.repo
sudo dnf makecache
```

**CentOS Stream 9 示例**（USTC）：

```bash
sudo tee /etc/yum.repos.d/centos-stream.repo >/dev/null <<'EOF'
[baseos]
name=CentOS Stream $releasever - BaseOS
baseurl=https://mirrors.ustc.edu.cn/centos-stream/$releasever-stream/BaseOS/$basearch/os/
gpgcheck=1
enabled=1
gpgkey=file:///etc/pki/rpm-gpg/RPM-GPG-KEY-centosofficial

[appstream]
name=CentOS Stream $releasever - AppStream
baseurl=https://mirrors.ustc.edu.cn/centos-stream/$releasever-stream/AppStream/$basearch/os/
gpgcheck=1
enabled=1
gpgkey=file:///etc/pki/rpm-gpg/RPM-GPG-KEY-centosofficial
EOF
sudo dnf makecache
```

**Fedora**（metalink 慢时手动指定）：

```bash
sudo sed -i.bak \
  -e 's|^metalink=|#metalink=|g' \
  -e 's|^#baseurl=|baseurl=|g' \
  -e 's|download.example|mirrors.ustc.edu.cn|g' \
  /etc/yum.repos.d/fedora*.repo
sudo dnf makecache
```

---

### apk — Alpine Linux

**配置文件**：`/etc/apk/repositories`

```bash
sudo cp /etc/apk/repositories /etc/apk/repositories.bak
sudo sed -i 's#https\?://dl-cdn.alpinelinux.org/alpine#https://mirrors.ustc.edu.cn/alpine#g' /etc/apk/repositories
sudo apk update
```

---

### pacman — Arch Linux / Manjaro

**配置文件**：`/etc/pacman.d/mirrorlist`

```bash
sudo cp /etc/pacman.d/mirrorlist /etc/pacman.d/mirrorlist.bak
sudo sed -i '1i Server = https://mirrors.ustc.edu.cn/archlinux/$repo/os/$arch' /etc/pacman.d/mirrorlist
sudo pacman -Syyu
```

---

### opkg — OpenWrt

**配置文件**：`/etc/opkg/distfeeds.conf`

```bash
cp /etc/opkg/distfeeds.conf /etc/opkg/distfeeds.conf.bak
sed -i 's_https\?://downloads.openwrt.org_https://mirrors.ustc.edu.cn/openwrt_g' /etc/opkg/distfeeds.conf
opkg update
```

> 部分镜像站只同步 release 分支，snapshots 需确认镜像站是否提供。

---

## 语言与构建工具

### Node.js — npm / pnpm / yarn

```bash
npm config set registry https://registry.npmmirror.com
pnpm config set registry https://registry.npmmirror.com
yarn config set registry https://registry.npmmirror.com
# Yarn Berry: yarn config set npmRegistryServer https://registry.npmmirror.com
```

**nvm / fnm 安装 Node 本体**：

```bash
export NVM_NODEJS_ORG_MIRROR=https://npmmirror.com/mirrors/node/
# fnm: export FNM_NODE_DIST_MIRROR=https://npmmirror.com/mirrors/node/
```

---

### Python — pip / uv

```bash
pip config set global.index-url https://pypi.tuna.tsinghua.edu.cn/simple
# uv 临时：UV_INDEX_URL=https://pypi.tuna.tsinghua.edu.cn/simple uv pip install <pkg>
```

**pyenv 编译安装 Python 本体**：

```bash
export PYTHON_BUILD_MIRROR_URL=https://mirrors.tuna.tsinghua.edu.cn/python/
```

---

### Go Modules

```bash
go env -w GOPROXY=https://goproxy.cn,direct
go env -w GOSUMDB=sum.golang.google.cn
```

**下载 Go 安装包**（非 go mod）：`https://mirrors.aliyun.com/golang/`

---

### Rust — rustup + Cargo

```bash
# rustup 安装/更新
export RUSTUP_DIST_SERVER=https://mirrors.ustc.edu.cn/rust-static
export RUSTUP_UPDATE_ROOT=https://mirrors.ustc.edu.cn/rust-static/rustup

# Cargo crates.io（需 cargo 1.68+ sparse 协议）
mkdir -p ~/.cargo
cat > ~/.cargo/config.toml <<'EOF'
[source.crates-io]
replace-with = "ustc"
[source.ustc]
registry = "sparse+https://mirrors.ustc.edu.cn/crates.io-index/"
EOF
```

---

### Java — Maven / Gradle

**Maven**（`~/.m2/settings.xml`）：

```xml
<mirror>
  <id>aliyunmaven</id>
  <mirrorOf>*</mirrorOf>
  <url>https://maven.aliyun.com/repository/public</url>
</mirror>
```

**Gradle**（`build.gradle` / `build.gradle.kts`）：

```groovy
repositories {
    maven { url 'https://maven.aliyun.com/repository/public' }
    mavenCentral()
}
```

---

### PHP — Composer

```bash
composer config -g repos.packagist composer https://mirrors.aliyun.com/composer/
```

---

### Ruby — RubyGems

```bash
gem sources --add https://mirrors.tuna.tsinghua.edu.cn/rubygems/ --remove https://rubygems.org/
```

---

### Conda / Miniconda

```bash
conda config --remove-key channels 2>/dev/null || true
conda config --add channels https://mirrors.ustc.edu.cn/anaconda/pkgs/main
conda config --add channels https://mirrors.ustc.edu.cn/anaconda/pkgs/r
conda config --add channels https://mirrors.ustc.edu.cn/anaconda/cloud/conda-forge
conda config --set show_channel_urls yes
```

**Miniconda 安装包**：`https://mirrors.ustc.edu.cn/anaconda/miniconda/`

---

### Flutter / Dart

```bash
export PUB_HOSTED_URL=https://pub.flutter-io.cn
export FLUTTER_STORAGE_BASE_URL=https://storage.flutter-io.cn
```

---

## 容器与 Kubernetes

### Docker Hub 镜像加速

写入 `/etc/docker/daemon.json` 后 `systemctl daemon-reload && systemctl restart docker`。**建议配置 2～3 个**，失败自动 fallback：

```json
{
  "registry-mirrors": [
    "https://docker.m.daocloud.io",
    "https://docker.1ms.run",
    "https://docker.1panel.live"
  ]
}
```

| 加速器 | 地址 | 备注 |
| --- | --- | --- |
| DaoCloud | `https://docker.m.daocloud.io` | 社区维护，较稳定 |
| 毫秒镜像 | `https://docker.1ms.run` | 社区维护 |
| 1Panel | `https://docker.1panel.live` | 社区维护 |
| 耗子面板 | `https://hub.rat.dev` | 社区维护 |
| 轩辕（免费） | `https://docker.xuanyuan.me` | 社区维护 |
| 腾讯云 | `https://mirror.ccs.tencentyun.com` | 仅腾讯云 CVM 内网优选 |
| 阿里云 | `https://<code>.mirror.aliyuncs.com` | 需登录阿里云控制台获取个人加速地址 |
| 高校 | `docker.mirrors.ustc.edu.cn` / `docker.nju.edu.cn` / `docker.mirrors.sjtug.sjtu.edu.cn` | 教育网 |

> `curl` 探测 `/v2/` 返回 **401** 表示 registry 可达（v2 API 正常）。公共加速器随时可能限速或停服。

**containerd**（k8s 节点）：在 `/etc/containerd/config.toml` 的 `[plugins."io.containerd.grpc.v1.cri".registry.mirrors]` 配置同等地址。

---

### Docker CE 软件包安装源（非 Hub 加速）

Debian/Ubuntu 安装 `docker-ce` 时，把 `download.docker.com` 替换为镜像站：

```bash
# 示例：USTC
sudo sed -i.bak 's|download.docker.com|mirrors.ustc.edu.cn/docker-ce|g' /etc/apt/sources.list.d/docker*.list
sudo apt update
```

---

### GHCR / GCR / Quay / K8s 镜像

`registry-mirrors` **不会**自动代理 `ghcr.io`、`gcr.io` 等，需显式替换拉取地址或自建 pull-through cache。

| 源 | 国内替代/代理 | 用法 |
| --- | --- | --- |
| `gcr.io` | `gcr.azk8s.cn` | `gcr.io/foo/bar` → `gcr.azk8s.cn/foo/bar` |
| `quay.io` | `quay.azk8s.cn` | 同上规则 |
| `k8s.gcr.io` / `registry.k8s.io` | `registry.aliyuncs.com/google_containers` | 见阿里云容器镜像服务文档 |
| `ghcr.io` | 自建 registry 代理 | 上游指向 `https://ghcr.io`，拉取时改域名 |

---

### Helm Charts

```bash
helm repo add stable https://mirrors.tuna.tsinghua.edu.cn/helm/stable
# 或阿里云：helm repo add stable https://mirrors.aliyun.com/kubernetes/charts/
helm repo update
```

---

## 源码、模型与大文件下载

### GitHub 加速（clone / release / raw / archive）

**⚠️ 使用前务必探测**（社区代理，随时可能停服）：

```bash
for p in ghproxy.net gh-proxy.com mirror.ghproxy.com github.akams.cn gh.llkk.cc hub.gitmirror.com ghfast.top; do
  code=$(curl -sL -o /dev/null -w "%{http_code}" --max-time 10 "https://$p/")
  echo "$p -> $code"
done
```

| 代理域名 | 说明 |
| --- | --- |
| `https://ghproxy.net/` | 老牌，支持断点续传 |
| `https://gh-proxy.com/` | clone/release/raw/archive 全覆盖 |
| `https://mirror.ghproxy.com/` | 移动/联通优化，大文件 |
| `https://github.akams.cn/` | API/Clone/Releases/Raw |
| `https://gh.llkk.cc/` | URL 前缀代理 |
| `https://hub.gitmirror.com/` | clone/release |
| `https://ghfast.top/` | 速度快，偶有维护 |

**用法**（`<代理>` 换探测通过的域名）：

```bash
curl -LO https://<代理>/https://github.com/<owner>/<repo>/releases/download/<tag>/<file>
git clone https://<代理>/https://github.com/<owner>/<repo>.git
git config --global url."https://<代理>/https://github.com/".insteadOf "https://github.com/"
```

**高校 GitHub Release 镜像**（仅收录热门项目，非全量）：
- TUNA：`https://mirrors.tuna.tsinghua.edu.cn/github-release/`
- USTC：`https://mirrors.ustc.edu.cn/github-release/`

> 公共代理仅支持公开仓库；私有仓库、push、带 token 请求禁止走第三方代理。

---

### Hugging Face 模型/数据集

**仅当前命令注入**（不要写入 shell profile 长期生效）：

```bash
HF_ENDPOINT=https://hf-mirror.com huggingface-cli download <model>
```

---

## macOS — Homebrew

```bash
export HOMEBREW_API_DOMAIN="https://mirrors.ustc.edu.cn/homebrew-bottles/api"
export HOMEBREW_BOTTLE_DOMAIN="https://mirrors.ustc.edu.cn/homebrew-bottles"
export HOMEBREW_BREW_GIT_REMOTE="https://mirrors.ustc.edu.cn/brew.git"
export HOMEBREW_CORE_GIT_REMOTE="https://mirrors.ustc.edu.cn/homebrew-core.git"
brew update
```

> 不同镜像站子路径不同，以对应 `/help/brew.git.html` 为准。

---

## 通用探测脚本

部署前在目标主机执行，快速筛选当前可用的镜像：

```bash
probe() {
  local name="$1" url="$2"
  local code
  code=$(curl -sL -o /dev/null -w "%{http_code}" --max-time 10 "$url" 2>/dev/null || echo "000")
  if [[ "$code" =~ ^(200|301|302|307|308|401)$ ]]; then
    echo "OK  $name ($code)"
  else
    echo "FAIL $name ($code)"
  fi
}

# 系统镜像站
probe tuna     "https://mirrors.tuna.tsinghua.edu.cn"
probe ustc     "https://mirrors.ustc.edu.cn"
probe aliyun   "https://mirrors.aliyun.com"

# Docker Hub 加速（401=可达）
probe daocloud "https://docker.m.daocloud.io/v2/"
probe 1ms      "https://docker.1ms.run/v2/"
probe 1panel   "https://docker.1panel.live/v2/"

# GitHub 代理
for p in ghproxy.net gh-proxy.com ghfast.top; do
  probe "gh-$p" "https://$p/"
done
```

---

## 其他地区

IP 归属非中国大陆时，直接使用各工具官方默认源即可，无需额外镜像。

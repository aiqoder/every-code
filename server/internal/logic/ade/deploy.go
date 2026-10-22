package ade

import (
	"time"

	"chengzi-agent-code-server/pkg/timex"
)

func localTime(t time.Time) timex.LocalTime { return timex.LocalTime(t) }

// DefaultPlaybook 默认发布剧本：releases/<时间戳>、current 软链、shared；
// 顺序为构建、上传、切换软链、健康检查；失败把 current 指回上一版。
const DefaultPlaybook = `# ADE 发布剧本
# 远程布局：<remoteDir>/releases/<时间戳>、<remoteDir>/current、<remoteDir>/shared
steps:
  - name: 构建
    local: |
      set -e
      # 按项目在本地执行构建，产物目录由 build_output 决定
      echo "build"
  - name: 上传
    upload: true
    description: 把产物上传到 releases/<时间戳>
  - name: 切换软链
    remote: |
      set -e
      ln -sfn releases/{{version}} current
  - name: 健康检查
    remote: |
      set -e
      curl -fsS "${HEALTH_URL:-http://127.0.0.1/}" >/dev/null
rollback: |
  set -e
  ln -sfn releases/{{previous}} current
`

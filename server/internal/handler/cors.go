package handler

import (
	"net/http"

	"chengzi-agent-code-server/internal/config"

	"github.com/zeromicro/go-zero/rest"
)

// ServerOptions 汇总服务端运行期选项。
func ServerOptions(c config.Config) []rest.RunOption {
	return withCors(c.Cors)
}

// withCors 按配置放行跨域，供安卓壳与桌面渲染进程等非同源客户端访问接口。
// 浏览器从本机页面访问 127.0.0.1 时会带私网预检，这里一并放行。
func withCors(c config.CorsConfig) []rest.RunOption {
	if !c.Enabled {
		return nil
	}
	origins := c.AllowedOrigins
	if len(origins) == 0 {
		origins = []string{"*"}
	}
	return []rest.RunOption{rest.WithCustomCors(func(header http.Header) {
		header.Set("Access-Control-Allow-Private-Network", "true")
	}, nil, origins...)}
}

package pathutil

import (
	"context"
	"fmt"
	"net/http"
	"strconv"

	"github.com/zeromicro/go-zero/rest/pathvar"
)

// Uint64 从 go-zero 路由上下文读取 path 参数并解析为 uint64。
func Uint64(ctx context.Context, name string) (uint64, error) {
	r := (&http.Request{}).WithContext(ctx)
	vars := pathvar.Vars(r)
	raw := vars[name]
	if raw == "" {
		return 0, fmt.Errorf("missing path param: %s", name)
	}
	id, err := strconv.ParseUint(raw, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid path param %s: %w", name, err)
	}
	return id, nil
}

package llmchat

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"syscall"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/openai/openai-go/v3"
)

// IsRetryable 判断 LLM 调用错误是否为可重试的瞬时错误。
//
// 可重试：429（限流）、408、409、5xx（含 Anthropic 529 overloaded）、
// 网络层错误（连接重置、对端提前关闭、超时）以及 SDK 自身的请求超时。
// 不可重试：调用方 ctx 取消，以及其余 4xx（鉴权失败、请求参数错误等，重试无意义）。
func IsRetryable(err error) bool {
	if err == nil {
		return false
	}
	// 用户/上层主动取消不重试；注意 DeadlineExceeded 不在这里排除，
	// 它可能来自 SDK 内部超时（可重试），也可能来自父 ctx（由调用方结合 ctx.Err() 判断）。
	if errors.Is(err, context.Canceled) {
		return false
	}
	// API 错误按 HTTP 状态码分类（两个 SDK 的错误类型都带 StatusCode 字段）。
	if code, ok := apiStatusCode(err); ok {
		switch {
		case code == http.StatusTooManyRequests,
			code == http.StatusRequestTimeout,
			code == http.StatusConflict:
			return true
		case code >= 500:
			return true
		default:
			return false
		}
	}
	// 非 API 错误（无状态码）：SDK 内部超时、网络抖动均可重试。
	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}
	var netErr net.Error
	if errors.As(err, &netErr) {
		return true
	}
	return errors.Is(err, io.EOF) ||
		errors.Is(err, io.ErrUnexpectedEOF) ||
		errors.Is(err, syscall.ECONNRESET) ||
		errors.Is(err, syscall.ECONNABORTED) ||
		errors.Is(err, syscall.EPIPE)
}

// apiStatusCode 从 SDK 错误中提取 HTTP 状态码；非 API 错误返回 ok=false。
func apiStatusCode(err error) (code int, ok bool) {
	var oaiErr *openai.Error
	if errors.As(err, &oaiErr) && oaiErr != nil && oaiErr.StatusCode != 0 {
		return oaiErr.StatusCode, true
	}
	var antErr *anthropic.Error
	if errors.As(err, &antErr) && antErr != nil && antErr.StatusCode != 0 {
		return antErr.StatusCode, true
	}
	return 0, false
}

package llmchat

import (
	"context"
	"errors"
	"fmt"
	"io"
	"testing"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/openai/openai-go/v3"
)

type fakeNetErr struct{ msg string }

func (e fakeNetErr) Error() string   { return e.msg }
func (e fakeNetErr) Timeout() bool   { return false }
func (e fakeNetErr) Temporary() bool { return true }

func TestIsRetryable(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want bool
	}{
		{"nil", nil, false},
		{"context canceled", context.Canceled, false},
		{"wrapped canceled", fmt.Errorf("stream: %w", context.Canceled), false},
		{"plain error", errors.New("bad request"), false},
		{"deadline exceeded", context.DeadlineExceeded, true},
		{"net error", fakeNetErr{"connection reset"}, true},
		{"wrapped net error", fmt.Errorf("stream: %w", fakeNetErr{"reset"}), true},
		{"unexpected EOF", fmt.Errorf("stream: %w", io.ErrUnexpectedEOF), true},
		// SDK API 错误：只读 StatusCode 字段，不调 Error()（其依赖非 nil 的 Request/Response）。
		{"openai 429", &openai.Error{StatusCode: 429}, true},
		{"openai 408", &openai.Error{StatusCode: 408}, true},
		{"openai 500", &openai.Error{StatusCode: 500}, true},
		{"openai 400", &openai.Error{StatusCode: 400}, false},
		{"openai 401", &openai.Error{StatusCode: 401}, false},
		{"anthropic 429", &anthropic.Error{StatusCode: 429}, true},
		{"anthropic 529 overloaded", &anthropic.Error{StatusCode: 529}, true},
		{"anthropic 403", &anthropic.Error{StatusCode: 403}, false},
	}
	for _, c := range cases {
		if got := IsRetryable(c.err); got != c.want {
			t.Errorf("%s: IsRetryable=%v, want %v", c.name, got, c.want)
		}
	}
}

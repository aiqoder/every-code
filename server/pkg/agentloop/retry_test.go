package agentloop

import (
	"testing"
	"time"
)

// 退避序列固定为 3s、6s、12s（封顶），保证 429 限流有足够的恢复窗口。
func TestRetryDelaySequence(t *testing.T) {
	want := []time.Duration{3 * time.Second, 6 * time.Second, 12 * time.Second, 12 * time.Second}
	for i, w := range want {
		if got := retryDelay(i + 1); got != w {
			t.Errorf("retryDelay(%d)=%v, want %v", i+1, got, w)
		}
	}
}

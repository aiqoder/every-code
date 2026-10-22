package agentloop

import (
	"context"
	"time"
)

// LLM 重试退避参数：3s 起步、每次翻倍、上限 12s（即 3s / 6s / 12s 序列）。
// 过短的退避对 429 限流没有意义——限流窗口通常以秒甚至分钟计。
const (
	retryBaseDelay = 3 * time.Second
	retryMaxDelay  = 12 * time.Second
)

// retrySleep 退避睡眠，可被 ctx 中断；声明为包级变量便于测试替换。
var retrySleep = func(ctx context.Context, d time.Duration) error {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}

// retryDelay 计算第 attempt 次重试（从 1 计）的退避时长：3s、6s、12s 封顶。
func retryDelay(attempt int) time.Duration {
	d := retryBaseDelay
	for i := 1; i < attempt; i++ {
		d *= 2
		if d >= retryMaxDelay {
			d = retryMaxDelay
			break
		}
	}
	return d
}

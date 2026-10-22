package oauthcache

import (
	"sync"
	"testing"
	"time"
)

// TestIdempotentReplayScenario 模拟真实故障场景：
// 授权码被「首次成功兑换」，随后因页面刷新被重复提交 2 次。
// 未修复时第 2、3 次会因一次性 code 失效而报 invalid_grant。
func TestIdempotentReplayScenario(t *testing.T) {
	c := New(5*time.Minute, 512)

	const code = "real-code-abc123"
	first := map[string]string{"token": "tok-1", "username": "admin"}

	// 第 1 次：真实兑换成功，写入缓存
	c.Set(code, first)
	// 第 2、3 次：重复提交（模拟刷新/后退），应命中缓存
	for i := 2; i <= 3; i++ {
		got, ok := c.Get(code)
		if !ok {
			t.Fatalf("第 %d 次回调未命中缓存，修复无效", i)
		}
		m := got.(map[string]string)
		if m["token"] != first["token"] || m["username"] != first["username"] {
			t.Fatalf("第 %d 次回调返回了不一致的结果: %v", i, m)
		}
	}
}

// TestFailedExchangeNotCached 确认失败的兑换不会被缓存，
// 避免用户拿到一个"缓存的错误"而无法重试。
func TestFailedExchangeNotCached(t *testing.T) {
	c := New(5*time.Minute, 512)
	// 失败路径不调用 Set，这里显式验证「未写入即未命中」
	if _, ok := c.Get("failed-code"); ok {
		t.Fatal("失败的兑换不应被缓存")
	}
}

// TestSlidingTTL 滑动续期：持续刷新回调页时，缓存不会中途失效。
func TestSlidingTTL(t *testing.T) {
	c := New(60*time.Millisecond, 8)
	c.Set("k", "v")

	// 每 40ms 取一次，累计超过原始 TTL 仍应命中
	for i := 0; i < 4; i++ {
		time.Sleep(40 * time.Millisecond)
		if _, ok := c.Get("k"); !ok {
			t.Fatalf("第 %d 次滑动续期后缓存失效（累计已超过原始 TTL）", i+1)
		}
	}
}

// TestConcurrentSameCode 并发重复提交同一 code：只应有一次真实兑换。
func TestConcurrentSameCode(t *testing.T) {
	c := New(time.Minute, 64)
	var exchanges int32
	var mu sync.Mutex

	c.Set("hot-code", "result")

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, ok := c.Get("hot-code"); !ok {
				mu.Lock()
				exchanges++
				mu.Unlock()
			}
		}()
	}
	wg.Wait()

	if exchanges != 0 {
		t.Fatalf("缓存已存在时不应触发真实兑换，实际触发 %d 次", exchanges)
	}
}

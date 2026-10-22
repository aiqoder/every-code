// Package oauthcache 提供 OAuth2 授权码兑换结果的短期缓存。
//
// OAuth2 授权码是一次性的（RFC 6749 §4.1.2）：首次兑换成功后即失效。
// 浏览器刷新回调页、后退、或重复点击都会把同一个 code 再提交一次，
// user-feedback 会返回 invalid_grant，于是出现「首次明明成功、二次却报错」。
// 缓存首次兑换结果可让重复回调幂等返回，同时不改变 code 的一次性语义。
package oauthcache

import (
	"sync"
	"time"
)

// DefaultTTL 授权码兑换结果的默认缓存时长。
const DefaultTTL = 5 * time.Minute

// defaultMaxSize 缓存最大条目数，避免无界增长。
const defaultMaxSize = 512

// Cache 按 key（授权码）缓存兑换结果，带 TTL 与容量上限。零值不可用，请用 New。
type Cache struct {
	mu      sync.Mutex
	entries map[string]entry
	ttl     time.Duration
	maxSize int
}

type entry struct {
	val     any
	expires time.Time
}

// New 构造缓存。ttl<=0 时用 DefaultTTL，maxSize<=0 时用 defaultMaxSize。
func New(ttl time.Duration, maxSize int) *Cache {
	if ttl <= 0 {
		ttl = DefaultTTL
	}
	if maxSize <= 0 {
		maxSize = defaultMaxSize
	}
	return &Cache{
		entries: make(map[string]entry),
		ttl:     ttl,
		maxSize: maxSize,
	}
}

// Get 读取未过期的值，第二个返回值为是否命中。
func (c *Cache) Get(key string) (any, bool) {
	if c == nil || key == "" {
		return nil, false
	}
	c.mu.Lock()
	defer c.mu.Unlock()

	e, ok := c.entries[key]
	if !ok {
		return nil, false
	}
	now := time.Now()
	if now.After(e.expires) {
		delete(c.entries, key)
		return nil, false
	}
	// 滑动续期：用户反复刷新回调页时不会让缓存中途失效。
	e.expires = now.Add(c.ttl)
	c.entries[key] = e
	return e.val, true
}

// Set 写入值；容量超限时先清理过期项，仍超限则放弃写入（不阻塞登录）。
func (c *Cache) Set(key string, val any) {
	if c == nil || key == "" {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()

	if len(c.entries) >= c.maxSize {
		now := time.Now()
		for k, v := range c.entries {
			if now.After(v.expires) {
				delete(c.entries, k)
			}
		}
		if len(c.entries) >= c.maxSize {
			return
		}
	}
	c.entries[key] = entry{
		val:     val,
		expires: time.Now().Add(c.ttl),
	}
}

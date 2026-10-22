package oauthcache

import (
	"testing"
	"time"
)

func TestSetGet(t *testing.T) {
	c := New(time.Minute, 4)
	if _, ok := c.Get("missing"); ok {
		t.Fatal("空缓存不应命中")
	}
	c.Set("k", "v")
	got, ok := c.Get("k")
	if !ok || got.(string) != "v" {
		t.Fatalf("期望命中 v，实际 got=%v ok=%v", got, ok)
	}
}

func TestExpired(t *testing.T) {
	c := New(20*time.Millisecond, 4)
	c.Set("k", "v")
	time.Sleep(40 * time.Millisecond)
	if _, ok := c.Get("k"); ok {
		t.Fatal("过期后不应命中")
	}
}

func TestEmptyKeyIgnored(t *testing.T) {
	c := New(time.Minute, 4)
	c.Set("", "v")
	if _, ok := c.Get(""); ok {
		t.Fatal("空 key 不应缓存")
	}
}

func TestCapacityEvictsExpired(t *testing.T) {
	c := New(10*time.Millisecond, 2)
	c.Set("a", 1)
	c.Set("b", 2)
	time.Sleep(30 * time.Millisecond)
	// 容量已满，但过期项应被清理，新条目可写入
	c.Set("c", 3)
	if _, ok := c.Get("c"); !ok {
		t.Fatal("清理过期项后应能写入新条目")
	}
}

func TestConcurrentAccess(t *testing.T) {
	c := New(time.Minute, 128)
	done := make(chan struct{})
	for i := 0; i < 16; i++ {
		go func(n int) {
			defer func() { done <- struct{}{} }()
			for j := 0; j < 50; j++ {
				c.Set("key", n)
				c.Get("key")
			}
		}(i)
	}
	for i := 0; i < 16; i++ {
		<-done
	}
}

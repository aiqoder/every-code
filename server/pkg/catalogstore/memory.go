package catalogstore

import (
	"context"
	"encoding/json"
	"strings"
	"sync"
	"time"

	"chengzi-agent-code-server/pkg/clienthub/protocol"
)

// memStore 是 Redis 缺席时的内存回退，供本机开发与单机部署使用。
// 语义与 Redis 实现一致：按 user-client 存一份 catalog，记录带 TTL，心跳续期。
// 进程重启即清空。
type memStore struct {
	mu      sync.RWMutex
	ttl     time.Duration
	entries map[string]memEntry
}

type memEntry struct {
	raw     []byte
	expires time.Time
}

func newMemStore(ttl time.Duration) *memStore {
	if ttl <= 0 {
		ttl = 90 * time.Second
	}
	return &memStore{ttl: ttl, entries: make(map[string]memEntry)}
}

// alive 读一条未过期的记录，顺手清理过期项。调用方需持有锁。
func (m *memStore) alive(key string, now time.Time) (memEntry, bool) {
	e, ok := m.entries[key]
	if !ok {
		return memEntry{}, false
	}
	if !e.expires.IsZero() && now.After(e.expires) {
		delete(m.entries, key)
		return memEntry{}, false
	}
	return e, true
}

func (m *memStore) put(key string, raw []byte, now time.Time) {
	m.entries[key] = memEntry{raw: raw, expires: now.Add(m.ttl)}
}

func (m *memStore) save(_ context.Context, userID uint64, clientID string, catalog protocol.CatalogPayload) error {
	raw, err := json.Marshal(catalog)
	if err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.put(Key(userID, clientID), raw, time.Now())
	return nil
}

func (m *memStore) load(_ context.Context, userID uint64, clientID string) (protocol.CatalogPayload, error) {
	var out protocol.CatalogPayload
	m.mu.RLock()
	e, ok := m.alive(Key(userID, clientID), time.Now())
	m.mu.RUnlock()
	if !ok {
		return out, ErrNotFound
	}
	if err := json.Unmarshal(e.raw, &out); err != nil {
		return out, err
	}
	_ = normalize(&out)
	return out, nil
}

func (m *memStore) touch(_ context.Context, userID uint64, clientID string) error {
	if strings.TrimSpace(clientID) == "" {
		return nil
	}
	now := time.Now()
	m.mu.Lock()
	defer m.mu.Unlock()
	key := Key(userID, clientID)
	if e, ok := m.alive(key, now); ok {
		m.put(key, e.raw, now)
	}
	return nil
}

func (m *memStore) deleteClient(_ context.Context, userID uint64, clientID string) error {
	if strings.TrimSpace(clientID) == "" {
		return nil
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.entries, Key(userID, clientID))
	return nil
}

func (m *memStore) close() error { return nil }

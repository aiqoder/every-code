package catalogstore

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"

	"chengzi-agent-code-server/pkg/clienthub/protocol"
)

// ErrNotFound 表示该客户端没有 catalog。
var ErrNotFound = errors.New("catalog not found")

// backend 抽象存储实现，Redis 与内存回退都满足它。
type backend interface {
	save(ctx context.Context, userID uint64, clientID string, catalog protocol.CatalogPayload) error
	load(ctx context.Context, userID uint64, clientID string) (protocol.CatalogPayload, error)
	touch(ctx context.Context, userID uint64, clientID string) error
	deleteClient(ctx context.Context, userID uint64, clientID string) error
	close() error
}

// Store 把客户端按 user-client 上报的能力清单存起来。
// 键：chengzi:catalog:{userId}:{clientId}
// 智能体人设不在这里，聊天时由服务端定义套用。
// 优先用 Redis；Redis 不可达时退回进程内存，保证本机单机也能跑起来。
type Store struct {
	be  backend
	ttl time.Duration
}

// New 连接 Redis。ttl 为记录过期时间，心跳时应续期。
func New(addr, password string, db int, ttl time.Duration) (*Store, error) {
	addr = strings.TrimSpace(addr)
	if addr == "" {
		return nil, fmt.Errorf("redis host is required")
	}
	if ttl <= 0 {
		ttl = 90 * time.Second
	}
	rdb := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := rdb.Ping(ctx).Err(); err != nil {
		_ = rdb.Close()
		return nil, fmt.Errorf("连接 Redis 失败: %w", err)
	}
	return &Store{be: &redisBackend{rdb: rdb, ttl: ttl}, ttl: ttl}, nil
}

// NewMemory 构造纯内存实现，Redis 不可用时的回退。
func NewMemory(ttl time.Duration) *Store {
	if ttl <= 0 {
		ttl = 90 * time.Second
	}
	return &Store{be: newMemStore(ttl), ttl: ttl}
}

// Close 关闭连接。
func (s *Store) Close() error {
	if s == nil || s.be == nil {
		return nil
	}
	return s.be.close()
}

// Key 返回一台客户端 catalog 的 Redis 键。
func Key(userID uint64, clientID string) string {
	return fmt.Sprintf("chengzi:catalog:%d:%s", userID, strings.TrimSpace(clientID))
}

// Save 覆盖写入一台客户端的 catalog，并刷新 TTL。
func (s *Store) Save(ctx context.Context, userID uint64, clientID string, catalog protocol.CatalogPayload) error {
	if s == nil || s.be == nil {
		return fmt.Errorf("catalog store is nil")
	}
	if strings.TrimSpace(clientID) == "" {
		return fmt.Errorf("clientId 不能为空")
	}
	if err := normalize(&catalog); err != nil {
		return err
	}
	return s.be.save(ctx, userID, clientID, catalog)
}

// Load 读取一台客户端的 catalog。不存在时返回 ErrNotFound。
func (s *Store) Load(ctx context.Context, userID uint64, clientID string) (protocol.CatalogPayload, error) {
	var out protocol.CatalogPayload
	if s == nil || s.be == nil {
		return out, fmt.Errorf("catalog store is nil")
	}
	clientID = strings.TrimSpace(clientID)
	if clientID == "" {
		return out, ErrNotFound
	}
	return s.be.load(ctx, userID, clientID)
}

// TouchClient 续期该客户端 catalog。键不存在时忽略。
func (s *Store) TouchClient(ctx context.Context, userID uint64, clientID string) error {
	if s == nil || s.be == nil || strings.TrimSpace(clientID) == "" {
		return nil
	}
	return s.be.touch(ctx, userID, clientID)
}

// DeleteClient 删除某客户端的 catalog。
func (s *Store) DeleteClient(ctx context.Context, userID uint64, clientID string) error {
	if s == nil || s.be == nil || strings.TrimSpace(clientID) == "" {
		return nil
	}
	return s.be.deleteClient(ctx, userID, clientID)
}

// ---------------------------------------------------------------------------
// Redis 实现
// ---------------------------------------------------------------------------

type redisBackend struct {
	rdb *redis.Client
	ttl time.Duration
}

func (r *redisBackend) save(ctx context.Context, userID uint64, clientID string, catalog protocol.CatalogPayload) error {
	raw, err := json.Marshal(catalog)
	if err != nil {
		return err
	}
	return r.rdb.Set(ctx, Key(userID, clientID), raw, r.ttl).Err()
}

func (r *redisBackend) load(ctx context.Context, userID uint64, clientID string) (protocol.CatalogPayload, error) {
	var out protocol.CatalogPayload
	raw, err := r.rdb.Get(ctx, Key(userID, clientID)).Bytes()
	if errors.Is(err, redis.Nil) {
		return out, ErrNotFound
	}
	if err != nil {
		return out, err
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return out, err
	}
	_ = normalize(&out)
	return out, nil
}

func (r *redisBackend) touch(ctx context.Context, userID uint64, clientID string) error {
	return r.rdb.Expire(ctx, Key(userID, clientID), r.ttl).Err()
}

func (r *redisBackend) deleteClient(ctx context.Context, userID uint64, clientID string) error {
	return r.rdb.Del(ctx, Key(userID, clientID)).Err()
}

func (r *redisBackend) close() error { return r.rdb.Close() }

func normalize(catalog *protocol.CatalogPayload) error {
	catalog.AgentID = strings.TrimSpace(catalog.AgentID)
	catalog.Name = strings.TrimSpace(catalog.Name)
	catalog.PageChannel = strings.TrimSpace(catalog.PageChannel)
	catalog.Workspace = strings.TrimSpace(catalog.Workspace)
	osInfo, err := NormalizeOsInfo(catalog.OsInfo)
	if err != nil {
		return err
	}
	catalog.OsInfo = osInfo
	if catalog.Tools == nil {
		catalog.Tools = []protocol.ToolMeta{}
	}
	if catalog.Skills == nil {
		catalog.Skills = []protocol.SkillMeta{}
	}
	return nil
}

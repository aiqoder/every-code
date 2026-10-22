package common

import (
	"context"
	"fmt"
	"strings"

	"chengzi-agent-code-server/internal/repository"
	"chengzi-agent-code-server/pkg/authutil"
)

// ResolveAPIKeyUserID 解析 API Key 对应用户 ID（user-feedback 用户 ID）。
func ResolveAPIKeyUserID(ctx context.Context, repo repository.ApiKeyRepository, rawKey string) (uint64, error) {
	rawKey = strings.TrimSpace(rawKey)
	if rawKey == "" {
		return 0, fmt.Errorf("empty api key")
	}
	key, err := repo.GetByHash(ctx, authutil.HashToken(rawKey))
	if err != nil {
		return 0, fmt.Errorf("invalid api key")
	}
	_ = repo.TouchLastUsed(ctx, key.ID)
	return key.UserID, nil
}

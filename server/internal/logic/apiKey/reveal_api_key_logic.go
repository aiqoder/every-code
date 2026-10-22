package apiKey

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"chengzi-agent-code-server/internal/middleware"
	"chengzi-agent-code-server/internal/svc"
	"chengzi-agent-code-server/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/gorm"
)

type RevealApiKeyLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewRevealApiKeyLogic(ctx context.Context, svcCtx *svc.ServiceContext) *RevealApiKeyLogic {
	return &RevealApiKeyLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *RevealApiKeyLogic) RevealApiKey(req *types.IdPathReq) (resp *types.ApiKeyRevealResp, err error) {
	key, err := l.svcCtx.ApiKeyRepo.GetByID(l.ctx, req.Id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("API Key 不存在")
		}
		return nil, err
	}
	if key.UserID != middleware.UserID(l.ctx) {
		return nil, fmt.Errorf("API Key 不存在")
	}
	if strings.TrimSpace(key.KeyPlain) == "" {
		return nil, fmt.Errorf("该密钥创建于旧版本，无法查看明文，请删除后重新创建")
	}
	return &types.ApiKeyRevealResp{Key: key.KeyPlain}, nil
}

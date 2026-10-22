package apiKey

import (
	"context"
	"fmt"

	"chengzi-agent-code-server/internal/middleware"
	"chengzi-agent-code-server/internal/svc"
	"chengzi-agent-code-server/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/gorm"
)

type DeleteApiKeyLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewDeleteApiKeyLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeleteApiKeyLogic {
	return &DeleteApiKeyLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *DeleteApiKeyLogic) DeleteApiKey(req *types.IdPathReq) error {
	if err := l.svcCtx.ApiKeyRepo.Delete(l.ctx, req.Id, middleware.UserID(l.ctx)); err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("API Key 不存在")
		}
		return err
	}
	return nil
}

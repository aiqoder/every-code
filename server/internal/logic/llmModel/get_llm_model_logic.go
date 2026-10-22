package llmModel

import (
	"context"
	"errors"
	"fmt"

	"chengzi-agent-code-server/internal/logic/common"
	"chengzi-agent-code-server/internal/logic/llmprovider"
	"chengzi-agent-code-server/internal/middleware"
	"chengzi-agent-code-server/internal/svc"
	"chengzi-agent-code-server/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/gorm"
)

type GetLlmModelLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetLlmModelLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetLlmModelLogic {
	return &GetLlmModelLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetLlmModelLogic) GetLlmModel(req *types.IdPathReq) (interface{}, error) {
	m, err := l.svcCtx.LlmModelRepo.GetByID(l.ctx, req.Id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("模型不存在")
		}
		return nil, err
	}
	userID := middleware.UserID(l.ctx)
	if !llmprovider.CanAccessModel(m, userID) {
		return nil, fmt.Errorf("模型不存在或无权访问")
	}
	return common.ToLlmModelVOFor(m, userID, false), nil
}

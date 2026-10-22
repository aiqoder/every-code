package admin

import (
	"context"

	"chengzi-agent-code-server/internal/logic/common"
	"chengzi-agent-code-server/internal/svc"
	"chengzi-agent-code-server/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type CreateLlmModelLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewCreateLlmModelLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CreateLlmModelLogic {
	return &CreateLlmModelLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *CreateLlmModelLogic) CreateLlmModel(req *types.LlmModelReq) (*common.LlmModelVO, error) {
	if _, err := ensurePlatformProvider(l.ctx, l.svcCtx.LlmProviderRepo, req.ProviderID); err != nil {
		return nil, err
	}
	m, err := BuildLlmModelEntity(l.ctx, l.svcCtx.LlmProviderRepo, ToLlmModelInput(req))
	if err != nil {
		return nil, err
	}
	if err := l.svcCtx.LlmModelRepo.Create(l.ctx, m); err != nil {
		return nil, err
	}
	if provider, err := l.svcCtx.LlmProviderRepo.GetByID(l.ctx, m.ProviderID); err == nil {
		m.Provider = provider
	}
	return common.ToLlmModelVO(m), nil
}

package llmModel

import (
	"context"
	"fmt"

	"chengzi-agent-code-server/internal/logic/admin"
	"chengzi-agent-code-server/internal/logic/common"
	"chengzi-agent-code-server/internal/logic/llmprovider"
	"chengzi-agent-code-server/internal/logic/pathutil"
	"chengzi-agent-code-server/internal/middleware"
	"chengzi-agent-code-server/internal/svc"
	"chengzi-agent-code-server/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type MutateLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewMutateLogic(ctx context.Context, svcCtx *svc.ServiceContext) *MutateLogic {
	return &MutateLogic{Logger: logx.WithContext(ctx), ctx: ctx, svcCtx: svcCtx}
}

func (l *MutateLogic) Create(req *types.LlmModelReq) (*common.LlmModelVO, error) {
	userID := middleware.UserID(l.ctx)
	if err := l.requireOwnProvider(req.ProviderID, userID); err != nil {
		return nil, err
	}
	m, err := admin.BuildLlmModelEntity(l.ctx, l.svcCtx.LlmProviderRepo, admin.ToLlmModelInput(req))
	if err != nil {
		return nil, err
	}
	if err := l.svcCtx.LlmModelRepo.Create(l.ctx, m); err != nil {
		return nil, err
	}
	if provider, err := l.svcCtx.LlmProviderRepo.GetByID(l.ctx, m.ProviderID); err == nil {
		m.Provider = provider
	}
	return common.ToLlmModelVOFor(m, userID, false), nil
}

func (l *MutateLogic) Update(req *types.LlmModelReq) (*common.LlmModelVO, error) {
	id, err := pathutil.Uint64(l.ctx, "id")
	if err != nil {
		return nil, err
	}
	userID := middleware.UserID(l.ctx)
	existing, err := l.svcCtx.LlmModelRepo.GetByID(l.ctx, id)
	if err != nil {
		return nil, fmt.Errorf("模型不存在")
	}
	if !llmprovider.CanEditModel(existing, userID) {
		return nil, fmt.Errorf("无权修改该模型")
	}
	if err := l.requireOwnProvider(req.ProviderID, userID); err != nil {
		return nil, err
	}
	built, err := admin.BuildLlmModelEntity(l.ctx, l.svcCtx.LlmProviderRepo, admin.ToLlmModelInput(req))
	if err != nil {
		return nil, err
	}
	existing.ProviderID = built.ProviderID
	existing.Name = built.Name
	existing.Attachment = built.Attachment
	existing.Reasoning = built.Reasoning
	existing.ReasoningOptions = built.ReasoningOptions
	existing.Temperature = built.Temperature
	existing.InputCapabilities = built.InputCapabilities
	existing.InputContextLimit = built.InputContextLimit
	existing.OutputContextLimit = built.OutputContextLimit
	existing.InputPrice = built.InputPrice
	existing.OutputPrice = built.OutputPrice
	existing.CachePrice = built.CachePrice
	if err := l.svcCtx.LlmModelRepo.Update(l.ctx, existing); err != nil {
		return nil, err
	}
	if provider, err := l.svcCtx.LlmProviderRepo.GetByID(l.ctx, existing.ProviderID); err == nil {
		existing.Provider = provider
	}
	return common.ToLlmModelVOFor(existing, userID, false), nil
}

func (l *MutateLogic) Delete(req *types.IdPathReq) error {
	existing, err := l.svcCtx.LlmModelRepo.GetByID(l.ctx, req.Id)
	if err != nil {
		return fmt.Errorf("模型不存在")
	}
	if !llmprovider.CanEditModel(existing, middleware.UserID(l.ctx)) {
		return fmt.Errorf("无权删除该模型")
	}
	return l.svcCtx.LlmModelRepo.Delete(l.ctx, existing.ID)
}

func (l *MutateLogic) requireOwnProvider(providerID, userID uint64) error {
	p, err := l.svcCtx.LlmProviderRepo.GetByID(l.ctx, providerID)
	if err != nil {
		return fmt.Errorf("供应商不存在")
	}
	if !llmprovider.CanEdit(p, userID) {
		return fmt.Errorf("只能在自己的供应商下维护模型")
	}
	return nil
}

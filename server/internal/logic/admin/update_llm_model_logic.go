package admin

import (
	"context"
	"errors"
	"fmt"

	"chengzi-agent-code-server/internal/logic/common"
	"chengzi-agent-code-server/internal/logic/pathutil"
	"chengzi-agent-code-server/internal/svc"
	"chengzi-agent-code-server/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/gorm"
)

type UpdateLlmModelLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewUpdateLlmModelLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateLlmModelLogic {
	return &UpdateLlmModelLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *UpdateLlmModelLogic) UpdateLlmModel(req *types.LlmModelReq) (*common.LlmModelVO, error) {
	id, err := pathutil.Uint64(l.ctx, "id")
	if err != nil {
		return nil, err
	}
	existing, err := l.svcCtx.LlmModelRepo.GetByID(l.ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("llm model not found")
		}
		return nil, err
	}
	if _, err := ensurePlatformProvider(l.ctx, l.svcCtx.LlmProviderRepo, req.ProviderID); err != nil {
		return nil, err
	}
	built, err := BuildLlmModelEntity(l.ctx, l.svcCtx.LlmProviderRepo, ToLlmModelInput(req))
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
	return common.ToLlmModelVO(existing), nil
}

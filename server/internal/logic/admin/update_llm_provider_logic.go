package admin

import (
	"context"
	"fmt"

	"chengzi-agent-code-server/internal/logic/pathutil"
	"chengzi-agent-code-server/internal/model"
	"chengzi-agent-code-server/internal/svc"
	"chengzi-agent-code-server/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type UpdateLlmProviderLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewUpdateLlmProviderLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateLlmProviderLogic {
	return &UpdateLlmProviderLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *UpdateLlmProviderLogic) UpdateLlmProvider(req *types.LlmProviderReq) (*model.LlmProvider, error) {
	id, err := pathutil.Uint64(l.ctx, "id")
	if err != nil {
		return nil, err
	}
	in := toLlmProviderInput(req)
	if in.Name == "" {
		return nil, fmt.Errorf("name is required")
	}
	provider, err := ensurePlatformProvider(l.ctx, l.svcCtx.LlmProviderRepo, id)
	if err != nil {
		return nil, err
	}
	provider.Name = in.Name
	provider.Website = in.Website
	provider.APIKey = in.APIKey
	provider.OpenAICompatibleBaseURL = in.OpenAICompatibleBaseURL
	provider.AnthropicBaseURL = in.AnthropicBaseURL
	if err := l.svcCtx.LlmProviderRepo.Update(l.ctx, provider); err != nil {
		return nil, err
	}
	return provider, nil
}

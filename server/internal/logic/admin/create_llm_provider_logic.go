package admin

import (
	"context"
	"fmt"

	"chengzi-agent-code-server/internal/model"
	"chengzi-agent-code-server/internal/svc"
	"chengzi-agent-code-server/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type CreateLlmProviderLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewCreateLlmProviderLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CreateLlmProviderLogic {
	return &CreateLlmProviderLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *CreateLlmProviderLogic) CreateLlmProvider(req *types.LlmProviderReq) (*model.LlmProvider, error) {
	in := toLlmProviderInput(req)
	if in.Name == "" {
		return nil, fmt.Errorf("name is required")
	}
	provider := &model.LlmProvider{
		UserID:                  nil,
		Name:                    in.Name,
		Website:                 in.Website,
		APIKey:                  in.APIKey,
		OpenAICompatibleBaseURL: in.OpenAICompatibleBaseURL,
		AnthropicBaseURL:        in.AnthropicBaseURL,
	}
	if err := l.svcCtx.LlmProviderRepo.Create(l.ctx, provider); err != nil {
		return nil, err
	}
	return provider, nil
}

package llmModel

import (
	"context"

	"chengzi-agent-code-server/internal/model"
	"chengzi-agent-code-server/internal/svc"
	"chengzi-agent-code-server/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type InputCapabilitiesLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewInputCapabilitiesLogic(ctx context.Context, svcCtx *svc.ServiceContext) *InputCapabilitiesLogic {
	return &InputCapabilitiesLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *InputCapabilitiesLogic) InputCapabilities() (resp *types.InputCapabilitiesResp, err error) {
	return &types.InputCapabilitiesResp{
		List: model.InputCapabilityOptions(),
	}, nil
}

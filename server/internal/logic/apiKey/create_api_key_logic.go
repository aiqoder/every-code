package apiKey

import (
	"context"
	"fmt"
	"strings"

	"chengzi-agent-code-server/internal/logic/common"
	"chengzi-agent-code-server/internal/middleware"
	"chengzi-agent-code-server/internal/model"
	"chengzi-agent-code-server/internal/svc"
	"chengzi-agent-code-server/internal/types"
	"chengzi-agent-code-server/pkg/authutil"

	"github.com/zeromicro/go-zero/core/logx"
)

type CreateApiKeyLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewCreateApiKeyLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CreateApiKeyLogic {
	return &CreateApiKeyLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *CreateApiKeyLogic) CreateApiKey(req *types.CreateApiKeyReq) (*common.ApiKeyCreatedVO, error) {
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return nil, fmt.Errorf("名称不能为空")
	}
	if len(name) > 64 {
		return nil, fmt.Errorf("名称过长")
	}
	secret, err := authutil.RandomHex(24)
	if err != nil {
		return nil, err
	}
	raw := "cz_" + secret
	prefix := raw
	if len(prefix) > 12 {
		prefix = prefix[:12]
	}
	key := &model.ApiKey{
		UserID:    middleware.UserID(l.ctx),
		Name:      name,
		KeyPrefix: prefix,
		KeyHash:   authutil.HashToken(raw),
		KeyPlain:  raw,
	}
	if err := l.svcCtx.ApiKeyRepo.Create(l.ctx, key); err != nil {
		return nil, err
	}
	return &common.ApiKeyCreatedVO{
		ID:        key.ID,
		Name:      key.Name,
		KeyPrefix: key.KeyPrefix,
		Key:       raw,
		CreatedAt: key.CreatedAt.Time().Format("2006-01-02 15:04:05"),
	}, nil
}

package client

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"chengzi-agent-code-server/internal/middleware"
	"chengzi-agent-code-server/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type RpcLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewRpcLogic(ctx context.Context, svcCtx *svc.ServiceContext) *RpcLogic {
	return &RpcLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *RpcLogic) Invoke(clientID, method string, params json.RawMessage) (json.RawMessage, error) {
	clientID = strings.TrimSpace(clientID)
	method = strings.TrimSpace(method)
	if clientID == "" {
		return nil, fmt.Errorf("缺少客户端")
	}
	if method == "" {
		return nil, fmt.Errorf("缺少方法")
	}
	return l.svcCtx.ClientHub.InvokeRPC(l.ctx, middleware.UserID(l.ctx), clientID, method, params)
}

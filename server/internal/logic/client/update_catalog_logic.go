package client

import (
	"context"
	"fmt"
	"strings"

	"chengzi-agent-code-server/internal/middleware"
	"chengzi-agent-code-server/internal/svc"
	"chengzi-agent-code-server/pkg/clienthub"
	"chengzi-agent-code-server/pkg/clienthub/protocol"

	"github.com/zeromicro/go-zero/core/logx"
)

type UpdateCatalogReq struct {
	ClientId    string               `json:"clientId"`
	Tools       []protocol.ToolMeta  `json:"tools,optional"`
	Skills      []protocol.SkillMeta `json:"skills,optional"`
	PageChannel string               `json:"pageChannel,optional"`
	OsInfo      string               `json:"osInfo"`
	Workspace   string               `json:"workspace"`
}

type UpdateCatalogLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewUpdateCatalogLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateCatalogLogic {
	return &UpdateCatalogLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *UpdateCatalogLogic) UpdateCatalog(req *UpdateCatalogReq) (*clienthub.ClientSummary, error) {
	clientID := strings.TrimSpace(req.ClientId)
	if clientID == "" {
		return nil, fmt.Errorf("clientId 不能为空")
	}
	userID := middleware.UserID(l.ctx)
	c, ok := l.svcCtx.ClientHub.Get(userID, clientID)
	if !ok || c == nil {
		return nil, fmt.Errorf("客户端未在线，请等待 WebSocket 连接后再注册")
	}
	catalog := protocol.CatalogPayload{
		Tools:       req.Tools,
		Skills:      req.Skills,
		PageChannel: strings.TrimSpace(req.PageChannel),
		OsInfo:      req.OsInfo,
		Workspace:   req.Workspace,
	}
	if err := l.svcCtx.ClientHub.UpdateCatalog(userID, clientID, catalog); err != nil {
		return nil, err
	}
	c, ok = l.svcCtx.ClientHub.Get(userID, clientID)
	if !ok || c == nil {
		return nil, fmt.Errorf("客户端未在线")
	}
	return &clienthub.ClientSummary{
		ClientID:  c.ClientID,
		Name:      c.Name,
		Workspace: c.Workspace,
	}, nil
}

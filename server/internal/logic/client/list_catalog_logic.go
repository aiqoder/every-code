package client

import (
	"context"
	"errors"
	"strings"

	"chengzi-agent-code-server/agents"
	"chengzi-agent-code-server/internal/agentcatalog"
	"chengzi-agent-code-server/internal/middleware"
	"chengzi-agent-code-server/internal/svc"
	"chengzi-agent-code-server/pkg/catalogstore"
	"chengzi-agent-code-server/pkg/clienthub/protocol"

	"github.com/zeromicro/go-zero/core/logx"
)

// CatalogAgent 服务端智能体套用到某台在线客户端后的投影。
type CatalogAgent struct {
	protocol.AgentMeta
	ClientID  string   `json:"clientId"`
	OsInfo    string   `json:"osInfo,omitempty"`
	Workspace string   `json:"workspace,omitempty"`
	Skills    []string `json:"skills,omitempty"`
}

// CatalogListResp 当前用户在线客户端上可运行的智能体。
type CatalogListResp struct {
	List []CatalogAgent `json:"list"`
}

type ListCatalogLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewListCatalogLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListCatalogLogic {
	return &ListCatalogLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// ListCatalog 用服务端智能体定义套用每台在线客户端的能力清单。
func (l *ListCatalogLogic) ListCatalog() (*CatalogListResp, error) {
	userID := middleware.UserID(l.ctx)
	out := make([]CatalogAgent, 0)
	store := l.svcCtx.ClientHub.CatalogStore()
	if store == nil {
		return &CatalogListResp{List: out}, nil
	}
	known := append(agents.All(), agents.Pipeline()...)
	for _, c := range l.svcCtx.ClientHub.List(userID) {
		cat, err := store.Load(l.ctx, userID, c.ClientID)
		if errors.Is(err, catalogstore.ErrNotFound) {
			continue
		}
		if err != nil {
			return nil, err
		}
		for _, agent := range known {
			view, err := agentcatalog.Materialize(l.ctx, l.svcCtx, agent.ID, cat)
			if err != nil {
				return nil, err
			}
			skillIDs := make([]string, 0, len(view.Skills))
			for _, sk := range view.Skills {
				if id := strings.TrimSpace(sk.ID); id != "" {
					skillIDs = append(skillIDs, id)
				}
			}
			out = append(out, CatalogAgent{
				AgentMeta: view.ToAgentMeta(),
				ClientID:  c.ClientID,
				OsInfo:    view.OsInfo,
				Workspace: view.Workspace,
				Skills:    skillIDs,
			})
		}
	}
	return &CatalogListResp{List: out}, nil
}

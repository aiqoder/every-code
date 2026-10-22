package agent

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"chengzi-agent-code-server/internal/agentcatalog"
	"chengzi-agent-code-server/internal/middleware"
	"chengzi-agent-code-server/internal/svc"
	"chengzi-agent-code-server/internal/types"
	"chengzi-agent-code-server/pkg/catalogstore"
	"chengzi-agent-code-server/pkg/skills"

	"github.com/zeromicro/go-zero/core/logx"
)

type ListSkillsReq struct {
	ClientId string `form:"clientId,optional"`
	AgentId  string `form:"agentId,optional"`
}

type ListSkillsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewListSkillsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListSkillsLogic {
	return &ListSkillsLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// ListSkills 返回在线客户端上报的技能。指定 agentId 时只留该智能体可用的技能。
func (l *ListSkillsLogic) ListSkills(req *ListSkillsReq) (resp *types.SkillListResp, err error) {
	if req == nil {
		req = &ListSkillsReq{}
	}
	filterClient := strings.TrimSpace(req.ClientId)
	filterAgent := strings.TrimSpace(req.AgentId)

	userID := middleware.UserID(l.ctx)
	store := l.svcCtx.ClientHub.CatalogStore()
	items := make([]skills.Skill, 0)
	exist := make(map[string]struct{})
	if store == nil {
		return &types.SkillListResp{List: items}, nil
	}

	for _, c := range l.svcCtx.ClientHub.List(userID) {
		if filterClient != "" && c.ClientID != filterClient {
			continue
		}
		cat, err := store.Load(l.ctx, userID, c.ClientID)
		if errors.Is(err, catalogstore.ErrNotFound) {
			continue
		}
		if err != nil {
			return nil, err
		}
		reported := cat.Skills
		if filterAgent != "" {
			view, err := agentcatalog.Materialize(l.ctx, l.svcCtx, filterAgent, cat)
			if err != nil {
				return nil, err
			}
			reported = view.Skills
		}
		for _, sk := range reported {
			id := strings.TrimSpace(sk.ID)
			if id == "" {
				continue
			}
			if _, ok := exist[id]; ok {
				continue
			}
			exist[id] = struct{}{}
			source := sk.Source
			if source == "" {
				source = "client"
			}
			items = append(items, skills.Skill{
				ID:          id,
				Name:        sk.Name,
				Description: sk.Description,
				Source:      source,
				Path:        sk.Path,
			})
		}
	}
	return &types.SkillListResp{List: items}, nil
}

// ParseListSkillsReq 从 HTTP 请求解析查询参数。
func ParseListSkillsReq(r *http.Request) *ListSkillsReq {
	q := r.URL.Query()
	return &ListSkillsReq{
		ClientId: q.Get("clientId"),
		AgentId:  q.Get("agentId"),
	}
}

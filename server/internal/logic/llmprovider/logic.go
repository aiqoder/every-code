package llmprovider

import (
	"context"
	"errors"
	"fmt"

	"gorm.io/gorm"

	"chengzi-agent-code-server/internal/logic/admin"
	"chengzi-agent-code-server/internal/logic/common"
	"chengzi-agent-code-server/internal/middleware"
	"chengzi-agent-code-server/internal/model"
	"chengzi-agent-code-server/internal/svc"
	"chengzi-agent-code-server/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type Logic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func New(ctx context.Context, svcCtx *svc.ServiceContext) *Logic {
	return &Logic{Logger: logx.WithContext(ctx), ctx: ctx, svcCtx: svcCtx}
}

func (l *Logic) List(req *types.LlmProviderListReq) (*types.LlmProviderListResp, error) {
	userID := middleware.UserID(l.ctx)
	if req.All {
		list, err := l.svcCtx.LlmProviderRepo.ListAllForUser(l.ctx, userID)
		if err != nil {
			return nil, err
		}
		return &types.LlmProviderListResp{List: toProviderVOs(list, userID)}, nil
	}
	page, pageSize := common.NormalizePage(req.Page, req.PageSize)
	offset := (page - 1) * pageSize
	list, total, err := l.svcCtx.LlmProviderRepo.ListForUser(l.ctx, userID, offset, pageSize)
	if err != nil {
		return nil, err
	}
	return &types.LlmProviderListResp{
		List:     toProviderVOs(list, userID),
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, nil
}

func (l *Logic) Get(req *types.IdPathReq) (*common.LlmProviderVO, error) {
	p, err := l.ownedOrPublic(req.Id)
	if err != nil {
		return nil, err
	}
	return common.ToLlmProviderVO(p, middleware.UserID(l.ctx), false), nil
}

func (l *Logic) Create(req *types.LlmProviderReq) (*common.LlmProviderVO, error) {
	userID := middleware.UserID(l.ctx)
	uid := userID
	p, err := buildProvider(req, &uid)
	if err != nil {
		return nil, err
	}
	if err := l.svcCtx.LlmProviderRepo.Create(l.ctx, p); err != nil {
		return nil, err
	}
	return common.ToLlmProviderVO(p, userID, false), nil
}

func (l *Logic) Update(req *types.LlmProviderReq, id uint64) (*common.LlmProviderVO, error) {
	p, err := l.mustOwn(id)
	if err != nil {
		return nil, err
	}
	if err := applyProvider(p, req); err != nil {
		return nil, err
	}
	if err := l.svcCtx.LlmProviderRepo.Update(l.ctx, p); err != nil {
		return nil, err
	}
	return common.ToLlmProviderVO(p, middleware.UserID(l.ctx), false), nil
}

func (l *Logic) Delete(req *types.IdPathReq) error {
	p, err := l.mustOwn(req.Id)
	if err != nil {
		return err
	}
	if err := l.svcCtx.LlmModelRepo.DeleteByProviderID(l.ctx, p.ID); err != nil {
		return err
	}
	return l.svcCtx.LlmProviderRepo.Delete(l.ctx, p.ID)
}

func (l *Logic) ListRemoteModels(req *types.IdPathReq) (*types.RemoteModelsResp, error) {
	if _, err := l.mustOwn(req.Id); err != nil {
		return nil, err
	}
	list, err := admin.FetchRemoteModels(l.ctx, l.svcCtx.LlmProviderRepo, req.Id)
	if err != nil {
		return nil, err
	}
	return &types.RemoteModelsResp{List: list}, nil
}

func (l *Logic) ownedOrPublic(id uint64) (*model.LlmProvider, error) {
	p, err := l.svcCtx.LlmProviderRepo.GetByID(l.ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("供应商不存在")
		}
		return nil, err
	}
	if !CanAccess(p, middleware.UserID(l.ctx)) {
		return nil, fmt.Errorf("供应商不存在或无权访问")
	}
	return p, nil
}

func (l *Logic) mustOwn(id uint64) (*model.LlmProvider, error) {
	p, err := l.ownedOrPublic(id)
	if err != nil {
		return nil, err
	}
	if !CanEdit(p, middleware.UserID(l.ctx)) {
		return nil, fmt.Errorf("无权修改该供应商")
	}
	return p, nil
}

func toProviderVOs(list []model.LlmProvider, userID uint64) []common.LlmProviderVO {
	vos := make([]common.LlmProviderVO, 0, len(list))
	for i := range list {
		vos = append(vos, *common.ToLlmProviderVO(&list[i], userID, false))
	}
	return vos
}

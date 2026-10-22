package brand

import (
	"context"

	"chengzi-agent-code-server/internal/config"
	"chengzi-agent-code-server/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

// Info 下发给各端的品牌。
type Info struct {
	Name string `json:"name"`
	Logo string `json:"logo"`
}

type GetBrandLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetBrandLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetBrandLogic {
	return &GetBrandLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// GetBrand 返回当前进程的品牌配置。空字段回填默认值。
func (l *GetBrandLogic) GetBrand() *Info {
	cfg := config.BrandConfig{}
	if l != nil && l.svcCtx != nil {
		cfg = l.svcCtx.Config.Brand
	}
	cfg.Normalize()
	return &Info{Name: cfg.Name, Logo: cfg.Logo}
}

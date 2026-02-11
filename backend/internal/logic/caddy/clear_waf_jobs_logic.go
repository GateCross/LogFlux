package caddy

import (
	"context"
	"fmt"

	"logflux/internal/svc"
	"logflux/internal/types"
	"logflux/model"

	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/gorm"
)

type ClearWafJobsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewClearWafJobsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ClearWafJobsLogic {
	return &ClearWafJobsLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ClearWafJobsLogic) ClearWafJobs() (resp *types.BaseResp, err error) {
	if err := l.svcCtx.DB.Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&model.WafUpdateJob{}).Error; err != nil {
		return nil, fmt.Errorf("clear waf jobs failed: %w", err)
	}

	return &types.BaseResp{Code: 200, Msg: "success"}, nil
}

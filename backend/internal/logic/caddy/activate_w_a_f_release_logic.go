package caddy

import (
	"context"
	"fmt"

	"logflux/internal/svc"
	"logflux/internal/types"
	"logflux/model"

	"github.com/zeromicro/go-zero/core/logx"
)

type ActivateWAFReleaseLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewActivateWAFReleaseLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ActivateWAFReleaseLogic {
	return &ActivateWAFReleaseLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ActivateWAFReleaseLogic) ActivateWAFRelease(req *types.WAFReleaseActivateReq) (resp *types.BaseResp, err error) {
	helper := newWAFLogicHelper(l.ctx, l.svcCtx, l.Logger)

	var release model.WAFRelease
	if err := helper.svcCtx.DB.First(&release, req.ID).Error; err != nil {
		return nil, fmt.Errorf("release not found")
	}

	job := helper.startJob(release.SourceID, release.ID, "activate", "manual")

	if err := helper.activateRelease(&release); err != nil {
		helper.markReleaseFailed(&release, err.Error())
		helper.finishJob(job, wafJobStatusFailed, err.Error(), release.ID)
		return nil, err
	}

	if err := helper.markReleaseActive(&release); err != nil {
		helper.finishJob(job, wafJobStatusFailed, err.Error(), release.ID)
		return nil, fmt.Errorf("mark active failed: %w", err)
	}

	helper.finishJob(job, wafJobStatusSuccess, "activate success", release.ID)
	return &types.BaseResp{Code: 200, Msg: "success"}, nil
}

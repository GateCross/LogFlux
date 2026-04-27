package caddy

import (
	"context"
	"errors"
	"fmt"

	"logflux/internal/svc"
	"logflux/internal/types"
	"logflux/model"

	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/gorm"
)

type DeleteWafSourceLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewDeleteWafSourceLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeleteWafSourceLogic {
	return &DeleteWafSourceLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *DeleteWafSourceLogic) DeleteWafSource(req *types.IDReq) (resp *types.BaseResp, err error) {
	var source model.WafSource
	if err := l.svcCtx.DB.WithContext(l.ctx).First(&source, req.ID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("源不存在")
		}
		return nil, fmt.Errorf("查询源失败: %w", err)
	}

	err = l.svcCtx.DB.WithContext(l.ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("source_id = ?", req.ID).Delete(&model.WafUpdateJob{}).Error; err != nil {
			return fmt.Errorf("删除源关联任务失败: %w", err)
		}

		if err := tx.Where("source_id = ?", req.ID).Delete(&model.WafRelease{}).Error; err != nil {
			return fmt.Errorf("删除源关联版本失败: %w", err)
		}

		if err := tx.Delete(&model.WafSource{}, req.ID).Error; err != nil {
			return fmt.Errorf("删除源失败: %w", err)
		}

		return nil
	})
	if err != nil {
		return nil, err
	}
	if l.svcCtx.WafScheduler != nil {
		l.svcCtx.WafScheduler.RemoveSource(req.ID)
	}

	return &types.BaseResp{Code: 200, Msg: "成功"}, nil
}

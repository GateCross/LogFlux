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
	if err := l.svcCtx.DB.First(&source, req.ID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("source not found")
		}
		return nil, fmt.Errorf("query source failed: %w", err)
	}

	err = l.svcCtx.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("source_id = ?", req.ID).Delete(&model.WafUpdateJob{}).Error; err != nil {
			return fmt.Errorf("delete source jobs failed: %w", err)
		}

		if err := tx.Where("source_id = ?", req.ID).Delete(&model.WafRelease{}).Error; err != nil {
			return fmt.Errorf("delete source releases failed: %w", err)
		}

		if err := tx.Delete(&model.WafSource{}, req.ID).Error; err != nil {
			return fmt.Errorf("delete source failed: %w", err)
		}

		return nil
	})
	if err != nil {
		return nil, err
	}
	if l.svcCtx.WafScheduler != nil {
		l.svcCtx.WafScheduler.RemoveSource(req.ID)
	}

	return &types.BaseResp{Code: 200, Msg: "success"}, nil
}

package caddy

import (
	"context"
	"fmt"

	"logflux/internal/svc"
	"logflux/internal/types"
	"logflux/model"

	"github.com/zeromicro/go-zero/core/logx"
)

type DeleteWafPolicyBindingLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewDeleteWafPolicyBindingLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeleteWafPolicyBindingLogic {
	return &DeleteWafPolicyBindingLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *DeleteWafPolicyBindingLogic) DeleteWafPolicyBinding(req *types.IDReq) (resp *types.BaseResp, err error) {
	defer func() {
		err = localizeWafPolicyError(err)
	}()

	if req == nil || req.ID == 0 {
		return nil, fmt.Errorf("policy binding id is required")
	}

	result := l.svcCtx.DB.Where("id = ?", req.ID).Delete(&model.WafPolicyBinding{})
	if result.Error != nil {
		return nil, fmt.Errorf("delete policy binding failed: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return nil, fmt.Errorf("policy binding not found")
	}

	return &types.BaseResp{Code: 200, Msg: "success"}, nil
}

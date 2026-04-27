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
		return nil, fmt.Errorf("策略绑定 ID 不能为空")
	}

	result := l.svcCtx.DB.WithContext(l.ctx).Where("id = ?", req.ID).Delete(&model.WafPolicyBinding{})
	if result.Error != nil {
		return nil, fmt.Errorf("删除策略绑定失败: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return nil, fmt.Errorf("策略绑定不存在")
	}

	return &types.BaseResp{Code: 200, Msg: "成功"}, nil
}

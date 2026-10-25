// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.1

package caddy

import (
	"context"

	"logflux/internal/svc"
	"logflux/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetCaddyGlobalOptionsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetCaddyGlobalOptionsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetCaddyGlobalOptionsLogic {
	return &GetCaddyGlobalOptionsLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetCaddyGlobalOptionsLogic) GetCaddyGlobalOptions() (resp *types.CaddyGlobalOptionsResp, err error) {
	value := loadCaddyGlobalOptions(l.svcCtx.DB)
	return globalOptionsValueToResp(value), nil
}

// globalOptionsValueToResp 把存储结构转为响应；Debug 指针缺省时输出 false。
func globalOptionsValueToResp(value CaddyGlobalOptionsValue) *types.CaddyGlobalOptionsResp {
	resp := &types.CaddyGlobalOptionsResp{
		AutoHttps:       value.AutoHttps,
		TrustedProxies:  value.TrustedProxies,
		Email:           value.Email,
		ExtraRawOptions: value.ExtraRawOptions,
	}
	if value.Debug != nil {
		resp.Debug = *value.Debug
	}
	return resp
}

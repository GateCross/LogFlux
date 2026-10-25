package caddy

import (
	"context"

	"logflux/internal/svc"
	"logflux/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type RefreshCertificatesLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewRefreshCertificatesLogic(ctx context.Context, svcCtx *svc.ServiceContext) *RefreshCertificatesLogic {
	return &RefreshCertificatesLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// RefreshCertificates 手动触发一次双源同步；serverId 为 0 时同步全部服务器。
func (l *RefreshCertificatesLogic) RefreshCertificates(req *types.CertificateRefreshReq) (resp *types.BaseResp, err error) {
	serverID := uint(0)
	if req != nil {
		serverID = req.ServerId
	}

	service := NewCertSyncService(l.ctx, l.svcCtx)
	if err := service.Sync(serverID); err != nil {
		return nil, err
	}
	return &types.BaseResp{Code: 0, Msg: "刷新完成"}, nil
}

package caddy

import (
	"context"
	"time"

	caddymodel "logflux/model/caddy"

	"logflux/internal/svc"
	"logflux/internal/types"
	"logflux/internal/xerr"

	"github.com/zeromicro/go-zero/core/logx"
)

type ListCertificatesLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewListCertificatesLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListCertificatesLogic {
	return &ListCertificatesLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// ListCertificates 返回唯一的证书库存。
// 有真值的行按当前时刻实时计算状态与剩余天数，避免读到每日同步前的旧状态。
func (l *ListCertificatesLogic) ListCertificates(req *types.CertificateListReq) (resp *types.CertificateListResp, err error) {
	if l.svcCtx.CertificateModel == nil {
		return nil, xerr.NewSystemErrorWith("证书模型未初始化")
	}

	rows, err := l.svcCtx.CertificateModel.List(l.ctx, caddymodel.CertificateQuery{ServerID: caddymodel.LegacyServerID})
	if err != nil {
		return nil, xerr.NewSystemErrorWith("查询证书库存失败")
	}

	now := time.Now()
	list := make([]types.CertificateItem, 0, len(rows))
	for i := range rows {
		row := rows[i]
		item := types.CertificateItem{
			ID:            row.ID,
			Domains:       row.Domains,
			PrimaryDomain: row.PrimaryDomain,
			Issuer:        row.Issuer,
			Source:        row.Source,
			StoragePath:   row.StoragePath,
			Status:        row.Status,
			DaysLeft:      0,
		}
		if !row.NotBefore.IsZero() {
			item.NotBefore = row.NotBefore.Format(time.RFC3339)
		}
		if !row.NotAfter.IsZero() {
			item.NotAfter = row.NotAfter.Format(time.RFC3339)
			status, daysLeft := certStatusAndDaysLeft(now, row.NotAfter)
			item.Status = status
			item.DaysLeft = daysLeft
		}
		if !row.LastSyncedAt.IsZero() {
			item.LastSyncedAt = row.LastSyncedAt.Format(time.RFC3339)
		}
		list = append(list, item)
	}

	return &types.CertificateListResp{List: list}, nil
}

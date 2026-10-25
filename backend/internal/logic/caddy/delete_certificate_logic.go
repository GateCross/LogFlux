package caddy

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"

	caddymodel "logflux/model/caddy"

	"logflux/internal/svc"
	"logflux/internal/types"
	"logflux/internal/xerr"

	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/gorm"
)

type DeleteCertificateLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewDeleteCertificateLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeleteCertificateLogic {
	return &DeleteCertificateLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// DeleteCertificate 删除库存行；上传证书顺带清理由本系统写入的文件，
// 扫描类证书只删库存行（文件由 Caddy 管理，下次同步会如实重新入账）。
func (l *DeleteCertificateLogic) DeleteCertificate(req *types.IDReq) (resp *types.BaseResp, err error) {
	if req == nil || req.ID == 0 {
		return nil, xerr.NewBusinessErrorWith("证书 ID 不能为空")
	}
	if l.svcCtx.CertificateModel == nil {
		return nil, xerr.NewSystemErrorWith("证书模型未初始化")
	}

	row, err := l.svcCtx.CertificateModel.FindByID(l.ctx, req.ID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, xerr.NewEnumError(xerr.NotFound)
		}
		return nil, xerr.NewSystemErrorWith("查询证书失败")
	}

	if row.Source == caddymodel.CertSourceUpload && row.StoragePath != "" {
		dir := filepath.Dir(filepath.Clean(row.StoragePath))
		// 仅清理上传目录内的文件，防止库存脏数据诱导误删
		if strings.HasPrefix(filepath.Clean(dir), filepath.Clean(certUploadBaseDir)+string(os.PathSeparator)) ||
			filepath.Clean(dir) == filepath.Clean(certUploadBaseDir) {
			if removeErr := os.RemoveAll(dir); removeErr != nil {
				l.Logger.Errorf("[证书中心] 清理上传证书文件失败: dir=%s err=%v", dir, removeErr)
			}
		}
	}

	if err := l.svcCtx.CertificateModel.DeleteByIDs(l.ctx, []uint{row.ID}); err != nil {
		return nil, xerr.NewSystemErrorWith("删除证书失败")
	}
	return &types.BaseResp{Code: 0, Msg: "删除成功"}, nil
}

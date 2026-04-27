package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"logflux/internal/ingest"
	"logflux/internal/svc"
	"logflux/internal/types"
	"logflux/internal/utils/logger"
	"logflux/internal/xerr"
	"logflux/model"

	"gorm.io/gorm"
)

// LogSourceService 负责日志源管理业务。
type LogSourceService struct {
	logger.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewLogSourceService(ctx context.Context, svcCtx *svc.ServiceContext) *LogSourceService {
	return &LogSourceService{
		Logger: logger.New(logger.ModuleLog).WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (s *LogSourceService) Add(req *types.LogSourceReq) (*types.BaseResp, error) {
	name := strings.TrimSpace(req.Name)
	path := strings.TrimSpace(req.Path)
	sourceType := strings.TrimSpace(req.Type)
	if sourceType == "" {
		sourceType = "caddy"
	}
	if name == "" {
		name = path
	}
	if path == "" {
		return nil, xerr.NewBusinessErrorWith("日志源路径不能为空")
	}
	scanInterval := req.ScanInterval
	if scanInterval < 0 {
		return nil, xerr.NewBusinessErrorWith("扫描间隔必须大于 0 秒")
	}
	if scanInterval <= 0 {
		scanInterval = ingest.DefaultScanIntervalSec()
	}

	source := &model.LogSource{
		Name:         name,
		Path:         path,
		Type:         sourceType,
		Enabled:      true,
		ScanInterval: scanInterval,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	if err := s.svcCtx.LogSourceModel.Create(s.ctx, source); err != nil {
		return nil, xerr.NewCodeErrorWithCause(xerr.ServerCommonError, "创建日志源失败", err)
	}
	s.svcCtx.Ingestor.StartWithInterval(source.Path, source.ScanInterval, source.Type)
	return baseResp("创建成功"), nil
}

func (s *LogSourceService) Delete(req *types.IDReq) (*types.BaseResp, error) {
	source, err := s.svcCtx.LogSourceModel.FindByID(s.ctx, req.ID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, xerr.NewBusinessErrorWith("日志源不存在")
		}
		return nil, xerr.NewCodeErrorWithCause(xerr.ServerCommonError, "查询日志源失败", err)
	}
	if err := s.svcCtx.LogSourceModel.DeleteByID(s.ctx, req.ID); err != nil {
		return nil, xerr.NewCodeErrorWithCause(xerr.ServerCommonError, "删除日志源失败", err)
	}
	if source.Path != "" {
		s.svcCtx.Ingestor.Stop(source.Path, source.Type)
	}
	return baseResp("删除成功"), nil
}

func (s *LogSourceService) List(req *types.LogSourceListReq) (*types.LogSourceListResp, error) {
	sources, total, err := s.svcCtx.LogSourceModel.List(s.ctx, req.Page, req.PageSize)
	if err != nil {
		return nil, xerr.NewCodeErrorWithCause(xerr.ServerCommonError, "查询日志源失败", err)
	}

	list := make([]types.LogSourceItem, 0, len(sources))
	for _, source := range sources {
		scanInterval := source.ScanInterval
		if scanInterval <= 0 {
			scanInterval = ingest.DefaultScanIntervalSec()
		}
		list = append(list, types.LogSourceItem{
			ID:           source.ID,
			Name:         source.Name,
			Path:         source.Path,
			Type:         source.Type,
			Enabled:      source.Enabled,
			ScanInterval: scanInterval,
			CreatedAt:    source.CreatedAt.Format("2006-01-02 15:04:05"),
		})
	}
	return &types.LogSourceListResp{List: list, Total: total}, nil
}

func (s *LogSourceService) Update(req *types.LogSourceUpdateReq) (*types.BaseResp, error) {
	source, err := s.svcCtx.LogSourceModel.FindByID(s.ctx, req.ID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, xerr.NewBusinessErrorWith("日志源不存在")
		}
		return nil, xerr.NewCodeErrorWithCause(xerr.ServerCommonError, "查询日志源失败", err)
	}

	oldPath := source.Path
	oldEnabled := source.Enabled
	if strings.TrimSpace(req.Name) != "" {
		source.Name = strings.TrimSpace(req.Name)
	}
	if req.Path != "" {
		path := strings.TrimSpace(req.Path)
		if path == "" {
			return nil, xerr.NewBusinessErrorWith("日志源路径不能为空")
		}
		source.Path = path
	}
	if req.ScanInterval < 0 {
		return nil, xerr.NewBusinessErrorWith("扫描间隔必须大于 0 秒")
	}
	if req.ScanInterval > 0 {
		source.ScanInterval = req.ScanInterval
	}
	source.Enabled = req.Enabled
	source.UpdatedAt = time.Now()

	if err := s.svcCtx.LogSourceModel.Save(s.ctx, source); err != nil {
		return nil, xerr.NewCodeErrorWithCause(xerr.ServerCommonError, "更新日志源失败", err)
	}

	if oldEnabled && (source.Path != oldPath || !source.Enabled) && oldPath != "" {
		s.svcCtx.Ingestor.Stop(oldPath, source.Type)
	}
	if source.Enabled && source.Path != "" {
		s.svcCtx.Ingestor.StartWithInterval(source.Path, source.ScanInterval, source.Type)
	}
	return baseResp("更新成功"), nil
}

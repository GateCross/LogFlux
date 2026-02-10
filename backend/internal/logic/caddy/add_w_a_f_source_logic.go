package caddy

import (
	"context"
	"fmt"
	"strings"

	"logflux/internal/svc"
	"logflux/internal/types"
	"logflux/model"

	"github.com/zeromicro/go-zero/core/logx"
)

type AddWAFSourceLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewAddWAFSourceLogic(ctx context.Context, svcCtx *svc.ServiceContext) *AddWAFSourceLogic {
	return &AddWAFSourceLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *AddWAFSourceLogic) AddWAFSource(req *types.WAFSourceReq) (resp *types.BaseResp, err error) {
	helper := newWAFLogicHelper(l.ctx, l.svcCtx, l.Logger)

	name := strings.TrimSpace(req.Name)
	if name == "" {
		return nil, fmt.Errorf("source name is required")
	}

	kind := normalizeWAFKind(req.Kind)
	if err := validateWAFKind(kind); err != nil {
		return nil, err
	}

	mode := normalizeWAFMode(req.Mode)
	if err := validateWAFMode(mode); err != nil {
		return nil, err
	}

	authType := normalizeWAFAuthType(req.AuthType)
	if err := validateWAFAuthType(authType); err != nil {
		return nil, err
	}

	sourceURL := strings.TrimSpace(req.Url)
	if mode == wafModeRemote && sourceURL == "" {
		return nil, fmt.Errorf("url is required for remote source")
	}

	meta, err := parseMetaJSON(req.Meta)
	if err != nil {
		return nil, err
	}

	source := &model.WAFSource{
		Name:         name,
		Kind:         kind,
		Mode:         mode,
		URL:          sourceURL,
		ChecksumURL:  strings.TrimSpace(req.ChecksumUrl),
		AuthType:     authType,
		AuthSecret:   strings.TrimSpace(req.AuthSecret),
		Schedule:     strings.TrimSpace(req.Schedule),
		Enabled:      true,
		AutoCheck:    true,
		AutoDownload: true,
		AutoActivate: false,
		Meta:         meta,
	}

	if helper.hasSourceBoolField("enabled") {
		source.Enabled = req.Enabled
	}
	if helper.hasSourceBoolField("autoCheck") {
		source.AutoCheck = req.AutoCheck
	}
	if helper.hasSourceBoolField("autoDownload") {
		source.AutoDownload = req.AutoDownload
	}
	if helper.hasSourceBoolField("autoActivate") {
		source.AutoActivate = req.AutoActivate
	}

	if err := helper.svcCtx.DB.Create(source).Error; err != nil {
		return nil, fmt.Errorf("create source failed: %w", err)
	}

	return &types.BaseResp{Code: 200, Msg: "success"}, nil
}

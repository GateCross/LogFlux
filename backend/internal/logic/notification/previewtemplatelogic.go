package notification

import (
	"context"

	"encoding/json"
	"logflux/internal/notification/template"
	"logflux/internal/svc"
	"logflux/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type PreviewTemplateLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewPreviewTemplateLogic(ctx context.Context, svcCtx *svc.ServiceContext) *PreviewTemplateLogic {
	return &PreviewTemplateLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *PreviewTemplateLogic) PreviewTemplate(req *types.PreviewTemplateReq) (resp *types.PreviewTemplateResp, err error) {
	var data interface{}
	if req.Data != "" {
		if err := json.Unmarshal([]byte(req.Data), &data); err != nil {
			return nil, err
		}
	} else {
		// Mock Data if not provided
		data = map[string]interface{}{
			"Type":    "system.test",
			"Level":   "info",
			"Message": "This is a preview message.",
			"Time":    "2023-10-01 12:00:00",
			"Data": map[string]interface{}{
				"key1": "value1",
				"key2": "value2",
			},
		}
	}

	// 使用 TemplateManager 的 RenderContent 静态方法
	rendered, err := template.RenderContent(req.Format, req.Content, data)
	if err != nil {
		return nil, err
	}

	return &types.PreviewTemplateResp{
		Content: rendered,
	}, nil
}

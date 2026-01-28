package caddy

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"

	"logflux/internal/svc"
	"logflux/internal/types"
	"logflux/model"

	"github.com/zeromicro/go-zero/core/logx"
)

type UpdateCaddyConfigLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewUpdateCaddyConfigLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateCaddyConfigLogic {
	return &UpdateCaddyConfigLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *UpdateCaddyConfigLogic) UpdateCaddyConfig(req *types.CaddyConfigUpdateReq) (resp *types.BaseResp, err error) {
	var server model.CaddyServer
	if err := l.svcCtx.DB.First(&server, req.ServerId).Error; err != nil {
		return nil, fmt.Errorf("server not found")
	}

	// Construct Caddy Admin API request
	// Usually POST /load to replace config, or POST /config/ to update specific path
	// Here we assume replacing the whole config or a part of it.
	// For simplicity, let's assume we are updating the whole config via /load
	// or adapting to the user requirement. The user said "modify caddy config".
	// Let's use /load endpoint which replaces the config.

	url := fmt.Sprintf("%s/load", server.Url)
	reqBody := bytes.NewBufferString(req.Config)

	httpReq, err := http.NewRequest("POST", url, reqBody)
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	// If remote auth is needed (not standard in default Caddy but possible via plugins or reverse proxy)
	if server.Token != "" {
		httpReq.Header.Set("Authorization", "Bearer "+server.Token)
	}

	client := &http.Client{}
	httpResp, err := client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer httpResp.Body.Close()

	if httpResp.StatusCode != 200 {
		body, _ := io.ReadAll(httpResp.Body)
		return nil, fmt.Errorf("caddy api error: %s", string(body))
	}

	return &types.BaseResp{
		Code: 200,
		Msg:  "success",
	}, nil
}

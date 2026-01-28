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

	// 1. Save Caddyfile to Database (Source of Truth)
	server.Config = req.Config
	if err := l.svcCtx.DB.Save(&server).Error; err != nil {
		l.Logger.Errorf("Failed to save config to DB: %v", err)
		return nil, fmt.Errorf("failed to save config to database")
	}

	// 2. Push to Caddy API
	// API currently expects JSON by default, but we want to send Caddyfile.
	// We use /load endpoint with Content-Type: text/caddyfile
	// Caddy will compile it to JSON on the fly.

	l.Logger.Infof("Pushing Caddyfile to server %s (ID: %d)", server.Name, server.ID)

	url := fmt.Sprintf("%s/load", server.Url)
	reqBody := bytes.NewBufferString(req.Config)

	httpReq, err := http.NewRequest("POST", url, reqBody)
	if err != nil {
		return nil, err
	}
	// Critical: Tell Caddy this is a Caddyfile, not JSON
	httpReq.Header.Set("Content-Type", "text/caddyfile")

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
		l.Logger.Errorf("Caddy API Error: %d - %s", httpResp.StatusCode, string(body))
		return nil, fmt.Errorf("caddy api error: %s", string(body))
	}

	l.Logger.Info("Caddy config updated successfully")
	return &types.BaseResp{
		Code: 200,
		Msg:  "success",
	}, nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

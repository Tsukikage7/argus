package command

import (
	"context"
)

// AlertEvent 告警事件（来自企微 Webhook 或监控系统）
type AlertEvent struct {
	AlertName   string `json:"alert_name"`
	Service     string `json:"service"`
	Severity    string `json:"severity"`
	Description string `json:"description"`
	Source      string `json:"source"` // prometheus / custom / wechat
}

// AlertEventHandler 处理告警事件
type AlertEventHandler struct {
	diagnoseHandler *DiagnoseHandler
}

// NewAlertEventHandler 创建告警事件处理器
func NewAlertEventHandler(dh *DiagnoseHandler) *AlertEventHandler {
	return &AlertEventHandler{diagnoseHandler: dh}
}

// Handle 处理告警事件 → 转化为诊断命令
func (h *AlertEventHandler) Handle(ctx context.Context, event AlertEvent) (string, error) {
	input := event.Description
	if input == "" {
		input = event.AlertName + " on " + event.Service
	}

	t, err := h.diagnoseHandler.Handle(ctx, DiagnoseCommand{
		Input:  input,
		Source: "webhook:" + event.Source,
	})
	if err != nil {
		return "", err
	}
	return t.ID, nil
}
